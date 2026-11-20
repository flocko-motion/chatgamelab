// package: events / SSE workshop event broker
// type:    logic
// job:     fans out workshop events to subscribed SSE clients keyed by workshop
// limits:  in-memory pub/sub only; no persistence or cross-process delivery
package events

import (
	"cgl/log"
	"sync"

	"github.com/google/uuid"
)

// EventType represents the type of workshop event
type EventType string

const (
	// WorkshopUpdated is emitted when workshop settings change
	WorkshopUpdated EventType = "workshop_updated"
	// GameCreated is emitted when a game is created in a workshop
	GameCreated EventType = "game_created"
	// GameUpdated is emitted when a game in a workshop is modified
	GameUpdated EventType = "game_updated"
	// GameDeleted is emitted when a game in a workshop is deleted
	GameDeleted EventType = "game_deleted"
	// MembersUpdated is emitted when a member joins or leaves a workshop
	MembersUpdated EventType = "members_updated"
	// Connected opens a subscriber's stream
	Connected EventType = "connected"
)

// WorkshopEvent is one update to a workshop, as its subscribers receive it.
// A game event names the game and who triggered it, so a client can ignore
// its own changes; the connected event names the workshop.
type WorkshopEvent struct {
	Type        EventType  `json:"type"`
	WorkshopID  *uuid.UUID `json:"workshopId,omitempty"`
	GameID      *uuid.UUID `json:"gameId,omitempty"`
	TriggeredBy *uuid.UUID `json:"triggeredBy,omitempty"`
}

// Broker manages SSE connections per workshop
type Broker struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan WorkshopEvent]struct{}
}

// Global broker instance
var globalBroker = &Broker{
	subscribers: make(map[uuid.UUID]map[chan WorkshopEvent]struct{}),
}

// GetBroker returns the global event broker
func GetBroker() *Broker {
	return globalBroker
}

// Subscribe creates a new event channel for a workshop
func (b *Broker) Subscribe(workshopID uuid.UUID) chan WorkshopEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan WorkshopEvent, 10)

	if b.subscribers[workshopID] == nil {
		b.subscribers[workshopID] = make(map[chan WorkshopEvent]struct{})
	}
	b.subscribers[workshopID][ch] = struct{}{}

	log.Debug("client subscribed to workshop events", "workshop_id", workshopID, "subscribers", len(b.subscribers[workshopID]))

	return ch
}

// Unsubscribe removes an event channel for a workshop
func (b *Broker) Unsubscribe(workshopID uuid.UUID, ch chan WorkshopEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[workshopID]; ok {
		delete(subs, ch)
		close(ch)

		// Clean up empty workshop entries
		if len(subs) == 0 {
			delete(b.subscribers, workshopID)
		}

		log.Debug("client unsubscribed from workshop events", "workshop_id", workshopID, "remaining", len(subs))
	}
}

// Publish sends an event to all subscribers of a workshop
func (b *Broker) Publish(workshopID uuid.UUID, event WorkshopEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[workshopID]
	if !ok || len(subs) == 0 {
		log.Debug("no subscribers for workshop event", "workshop_id", workshopID, "event", event.Type)
		return
	}

	log.Debug("publishing workshop event", "workshop_id", workshopID, "event", event.Type, "subscribers", len(subs))

	for ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, skip this subscriber
			log.Debug("subscriber channel full, skipping", "workshop_id", workshopID)
		}
	}
}

// PublishWorkshopUpdated is a convenience method to publish a workshop_updated event
func (b *Broker) PublishWorkshopUpdated(workshopID uuid.UUID) {
	b.Publish(workshopID, WorkshopEvent{Type: WorkshopUpdated})
}

// PublishGameCreated publishes a game_created event with game ID and creator ID
// The triggeredBy field allows clients to ignore events they triggered themselves
func (b *Broker) PublishGameCreated(workshopID uuid.UUID, gameID uuid.UUID, triggeredBy uuid.UUID) {
	b.Publish(workshopID, gameEvent(GameCreated, gameID, triggeredBy))
}

// PublishGameUpdated publishes a game_updated event with game ID and modifier ID
func (b *Broker) PublishGameUpdated(workshopID uuid.UUID, gameID uuid.UUID, triggeredBy uuid.UUID) {
	b.Publish(workshopID, gameEvent(GameUpdated, gameID, triggeredBy))
}

// PublishGameDeleted publishes a game_deleted event with game ID and deleter ID
func (b *Broker) PublishGameDeleted(workshopID uuid.UUID, gameID uuid.UUID, triggeredBy uuid.UUID) {
	b.Publish(workshopID, gameEvent(GameDeleted, gameID, triggeredBy))
}

// PublishMembersUpdated is a convenience method to publish a members_updated event
func (b *Broker) PublishMembersUpdated(workshopID uuid.UUID) {
	b.Publish(workshopID, WorkshopEvent{Type: MembersUpdated})
}

// gameEvent is an event about a game in a workshop.
func gameEvent(t EventType, gameID, triggeredBy uuid.UUID) WorkshopEvent {
	return WorkshopEvent{Type: t, GameID: &gameID, TriggeredBy: &triggeredBy}
}
