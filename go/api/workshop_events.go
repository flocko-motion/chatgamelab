// package: api / schemaf endpoints
// type:    logic
// job:     streams real-time workshop updates to clients over Server-Sent Events
// limits:  the events themselves come from events; schemaf writes the stream
package api

import (
	"context"

	"cgl/events"
	"cgl/log"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// WorkshopEvent is one update to a workshop; its type tells them apart.
type WorkshopEvent = events.WorkshopEvent

// WorkshopEventsReq names the workshop to follow.
type WorkshopEventsReq struct {
	ID string `path:"id" json:"-"`
}

// WorkshopEventsEndpoint streams a workshop's updates as Server-Sent Events.
//
// The stream opens with a "connected" event carrying the workshop id, then
// sends each update, until the client disconnects.
type WorkshopEventsEndpoint struct{}

func (WorkshopEventsEndpoint) Method() string             { return "GET" }
func (WorkshopEventsEndpoint) Path() string               { return "/api/workshops/{id}/events" }
func (WorkshopEventsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (WorkshopEventsEndpoint) Stream(ctx context.Context, req WorkshopEventsReq) (<-chan WorkshopEvent, error) {
	if _, err := caller(ctx); err != nil {
		return nil, err
	}
	workshopID, err := parseIDValidation(req.ID, "Invalid workshop ID")
	if err != nil {
		return nil, err
	}
	return workshopEvents(ctx, workshopID), nil
}

// workshopEvents is a workshop's updates, opened by a connected event, until
// ctx ends.
func workshopEvents(ctx context.Context, workshopID uuid.UUID) <-chan WorkshopEvent {
	broker := events.GetBroker()
	sub := broker.Subscribe(workshopID)
	out := make(chan WorkshopEvent)
	go func() {
		defer close(out)
		defer broker.Unsubscribe(workshopID, sub)
		log.Debug("SSE connection established", "workshop_id", workshopID)
		send := func(e WorkshopEvent) bool {
			select {
			case out <- e:
				return true
			case <-ctx.Done():
				return false
			}
		}
		if !send(WorkshopEvent{Type: events.Connected, WorkshopID: &workshopID}) {
			return
		}
		for {
			select {
			case e, ok := <-sub:
				if !ok || !send(e) {
					return
				}
			case <-ctx.Done():
				log.Debug("SSE connection closed by client", "workshop_id", workshopID)
				return
			}
		}
	}()
	return out
}
