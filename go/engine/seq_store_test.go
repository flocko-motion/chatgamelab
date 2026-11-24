package engine_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"engine"
	"engine/internal/adapters/mock"
)

// seqStore numbers events as the platform's table does: the next number is
// read, then written, and a number taken in between refuses the event.
type seqStore struct {
	engine.MemoryPersist
	mu   sync.Mutex
	seqs map[string]int
}

func (s *seqStore) SaveEvent(ctx context.Context, id string, e engine.SessionEvent) error {
	s.mu.Lock()
	if s.seqs == nil {
		s.seqs = map[string]int{}
	}
	seq := s.seqs[id] + 1
	s.mu.Unlock()
	time.Sleep(2 * time.Millisecond) // the round trip between reading and writing
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seqs[id] >= seq {
		return errors.New("duplicate key")
	}
	s.seqs[id] = seq
	return s.MemoryPersist.SaveEvent(ctx, id, e)
}

// What the player saw is stored whole, the picture and every line, though
// the journal is stored beside it at the same moments.
func TestHistoryIsStoredBesideTheJournal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &seqStore{}
	first, err := engine.Launch(ctx, engine.SessionSpec{
		Subgenre: engine.SubgenreNPCLive, ID: "both", Persist: store,
		Scenario: "You keep a bridge.", GalleryImage: mock.GalleryImage("bridge"),
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	if err := awaitConnection(ctx, first); err != nil {
		t.Fatal(err)
	}
	waitForTurn(t, first, "")
	shown := first.History()
	first.Close()

	stored := store.Events("both")
	for _, stream := range []string{"image", "text"} {
		var want, got []string
		for _, e := range shown {
			if e.Stream == stream {
				want = append(want, e.Value)
			}
		}
		for _, e := range stored {
			if e.Stream == stream {
				got = append(got, e.Value)
			}
		}
		if len(got) != len(want) {
			t.Errorf("%s: shown %d, stored %d", stream, len(want), len(got))
		}
	}
}
