package events

import (
	"cgl/log"
	"fmt"
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
)

// Event represents an SSE event to be sent to clients
type Event struct {
	Type EventType `json:"type"`
	Data string    `json:"data,omitempty"`
}

// Broker manages SSE connections per workshop
type Broker struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan Event]struct{}
}

// Global broker instance
var globalBroker = &Broker{
	subscribers: make(map[uuid.UUID]map[chan Event]struct{}),
}

// GetBroker returns the global event broker
func GetBroker() *Broker {
	return globalBroker
}

// Subscribe creates a new event channel for a workshop
func (b *Broker) Subscribe(workshopID uuid.UUID) chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 10)

	if b.subscribers[workshopID] == nil {
		b.subscribers[workshopID] = make(map[chan Event]struct{})
	}
	b.subscribers[workshopID][ch] = struct{}{}

	log.Debug("client subscribed to workshop events", "workshop_id", workshopID, "subscribers", len(b.subscribers[workshopID]))

	return ch
}

// Unsubscribe removes an event channel for a workshop
func (b *Broker) Unsubscribe(workshopID uuid.UUID, ch chan Event) {
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
func (b *Broker) Publish(workshopID uuid.UUID, event Event) {
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
	b.Publish(workshopID, Event{Type: WorkshopUpdated})
}

// PublishGameCreated publishes a game_created event with game ID and creator ID
// The triggeredBy field allows clients to ignore events they triggered themselves
func (b *Broker) PublishGameCreated(workshopID uuid.UUID, gameID uuid.UUID, triggeredBy uuid.UUID) {
	data := fmt.Sprintf(`{"gameId":"%s","triggeredBy":"%s"}`, gameID, triggeredBy)
	b.Publish(workshopID, Event{Type: GameCreated, Data: data})
}

// PublishGameUpdated publishes a game_updated event with game ID and modifier ID
func (b *Broker) PublishGameUpdated(workshopID uuid.UUID, gameID uuid.UUID, triggeredBy uuid.UUID) {
	data := fmt.Sprintf(`{"gameId":"%s","triggeredBy":"%s"}`, gameID, triggeredBy)
	b.Publish(workshopID, Event{Type: GameUpdated, Data: data})
}

// PublishGameDeleted publishes a game_deleted event with game ID and deleter ID
func (b *Broker) PublishGameDeleted(workshopID uuid.UUID, gameID uuid.UUID, triggeredBy uuid.UUID) {
	data := fmt.Sprintf(`{"gameId":"%s","triggeredBy":"%s"}`, gameID, triggeredBy)
	b.Publish(workshopID, Event{Type: GameDeleted, Data: data})
}
