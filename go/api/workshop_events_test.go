package api

import (
	"context"
	"testing"
	"time"

	"cgl/events"

	"github.com/google/uuid"
)

func TestWorkshopEventStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workshopID, gameID, by := uuid.New(), uuid.New(), uuid.New()

	stream := workshopEvents(ctx, workshopID)

	first := next(t, stream)
	if first.Type != events.Connected || first.WorkshopID == nil || *first.WorkshopID != workshopID {
		t.Fatalf("first event %+v, want connected with the workshop id", first)
	}

	events.GetBroker().PublishGameUpdated(workshopID, gameID, by)
	got := next(t, stream)
	if got.Type != events.GameUpdated || got.GameID == nil || *got.GameID != gameID || *got.TriggeredBy != by {
		t.Errorf("event %+v, want game_updated for the game", got)
	}

	cancel()
	select {
	case _, open := <-stream:
		if open {
			t.Error("stream sent after its ctx ended")
		}
	case <-time.After(time.Second):
		t.Error("stream did not end with its ctx")
	}
}

func next(t *testing.T, stream <-chan WorkshopEvent) WorkshopEvent {
	t.Helper()
	select {
	case e := <-stream:
		return e
	case <-time.After(time.Second):
		t.Fatal("no event")
		return WorkshopEvent{}
	}
}
