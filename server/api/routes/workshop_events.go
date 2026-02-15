package routes

import (
	"cgl/api/httpx"
	"cgl/events"
	"cgl/log"
	"cgl/obj"
	"fmt"
	"net/http"
	"time"
)

// WorkshopEvents godoc
//
//	@Summary		Subscribe to workshop events (SSE)
//	@Description	Server-Sent Events endpoint for real-time workshop updates. Supports token via query param for EventSource compatibility.
//	@Tags			workshops
//	@Produce		text/event-stream
//	@Param			id		path	string	true	"Workshop ID"
//	@Param			token	query	string	false	"Auth token (for EventSource which cannot send headers)"
//	@Success		200	{string}	string	"SSE stream"
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id}/events [get]
func WorkshopEvents(w http.ResponseWriter, r *http.Request) {
	// Note: Auth is handled by the middleware, but we verify user exists
	// The token query param is processed by WorkshopEventsWithTokenParam wrapper

	workshopID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get the flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Subscribe to workshop events
	broker := events.GetBroker()
	eventChan := broker.Subscribe(workshopID)
	defer broker.Unsubscribe(workshopID, eventChan)

	log.Debug("SSE connection established", "workshop_id", workshopID)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"workshopId\":\"%s\"}\n\n", workshopID)
	flusher.Flush()

	// Heartbeat keeps the connection alive through proxies (nginx, Cloudflare, etc.)
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Stream events until client disconnects
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				return
			}
			if event.Data != "" {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
			} else {
				fmt.Fprintf(w, "event: %s\ndata: {}\n\n", event.Type)
			}
			flusher.Flush()

		case <-heartbeat.C:
			// SSE comment line — ignored by EventSource but resets proxy idle timers
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			log.Debug("SSE connection closed by client", "workshop_id", workshopID)
			return
		}
	}
}
