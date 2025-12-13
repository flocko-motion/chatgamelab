package endpoints

import (
	"cgl/api/handler"
	"cgl/game/stream"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// MessageStream is the SSE endpoint for streaming message content
var MessageStream = handler.NewSSEEndpoint(
	"/api/messages/{id:uuid}/stream",
	true, // public - auth handled separately if needed
	getMessageStream,
)

// GET /api/messages/{id}/stream - SSE endpoint for streaming message content
func getMessageStream(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	messageID, err := uuid.Parse(pathParams["id"])
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	// Lookup the stream
	registry := stream.Get()
	s := registry.Lookup(messageID)
	if s == nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream chunks to client
	for chunk := range s.Chunks {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if chunk.Done {
			break
		}
	}

	// Cleanup
	registry.Remove(messageID)
}
