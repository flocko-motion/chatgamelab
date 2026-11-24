package engine_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"engine"
	"engine/internal/genre"
)

// A duo is two conversations in one scene: the topology names both seats,
// the browser connects to each, and both run.
func TestDuoOpensTwoConversations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:      engine.GenreNPCDuo,
		Scenario:   "A grumpy lighthouse keeper." + genre.DuoSeparator + "A cheerful tax inspector.",
		InitPrompt: "A stormy night in the lighthouse.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if seats := s.Topology().Seats; !slices.Equal(seats, []string{"", genre.SeatPartner}) {
		t.Fatalf("seats = %q, want the first and the partner", seats)
	}

	working := map[string]bool{}
	invited := false
	deadline := time.After(10 * time.Second)
	for len(working) < 2 {
		select {
		case e := <-s.Events():
			switch e.Stream {
			case "connect":
				if invited {
					continue
				}
				invited = true
				for _, seat := range s.Topology().Seats {
					if _, err := s.Connect(ctx, "v=0 mock offer", seat); err != nil {
						t.Fatalf("connect seat %q: %v", seat, err)
					}
				}
			case "state":
				for _, node := range []string{"live-a", "live-b"} {
					if e.Value == `{"node":"`+node+`","phase":"working"}` {
						working[node] = true
					}
				}
			}
		case <-deadline:
			t.Fatalf("not both conversations ran: %v (invited %t)", working, invited)
		}
	}

	if _, err := s.Connect(ctx, "v=0 mock offer", "c"); err == nil {
		t.Error("a seat the duo does not have was connected")
	}
}

// Pausing the scene lets both conversations go, and the pause is reported.
func TestDuoPausesAsOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCDuo,
		Scenario: "A knight." + genre.DuoSeparator + "A dragon.",
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	open := map[string]bool{}
	deadline := time.After(10 * time.Second)
	paused := false
	for !paused {
		select {
		case e := <-s.Events():
			switch e.Stream {
			case "connect":
				if len(open) > 0 {
					continue
				}
				for _, seat := range s.Topology().Seats {
					if _, err := s.Connect(ctx, "v=0 mock offer", seat); err != nil {
						t.Fatalf("connect seat %q: %v", seat, err)
					}
				}
			case "state":
				for _, node := range []string{"live-a", "live-b"} {
					if e.Value == `{"node":"`+node+`","phase":"working"}` {
						open[node] = true
					}
					if e.Value == `{"node":"`+node+`","phase":"ready"}` && open[node] {
						open[node] = false
					}
				}
				if open["live-a"] && open["live-b"] {
					if err := s.Pause(); err != nil {
						t.Fatal(err)
					}
				}
			case "pause":
				paused = true
			}
		case <-deadline:
			t.Fatal("the scene was never paused")
		}
	}
	// Either conversation may report first; the other follows at once.
	settle := time.Now().Add(2 * time.Second)
	for s.Snapshot().Conversation != "paused" && time.Now().Before(settle) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := s.Snapshot().Conversation; got != "paused" {
		t.Errorf("conversation = %q after a pause, want paused", got)
	}
}

func TestSplitDuo(t *testing.T) {
	a, b := genre.SplitDuo("first one" + genre.DuoSeparator + "second one")
	if a != "first one" || b != "second one" {
		t.Errorf("split = %q, %q", a, b)
	}
	a, b = genre.SplitDuo("only one")
	if a != "only one" || b != "" {
		t.Errorf("split of one = %q, %q", a, b)
	}
}

// A scene on the mock platform is cast from the mock's own tagged voices,
// each character in the gender the game chose and the player's language.
func TestDuoIsCastFromThePlatformsVoices(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCDuo,
		Scenario: "A knight." + genre.DuoSeparator + "A dragon.",
		Language: "de",
		Voices:   map[string]engine.VoiceChoice{"a": engine.VoiceFemale, "b": engine.VoiceMale},
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	deadline := time.After(10 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream != "cast" {
				continue
			}
			if !strings.Contains(e.Value, `"voiceA":"mock-greta"`) || !strings.Contains(e.Value, `"voiceB":"mock-hans"`) {
				t.Errorf("cast %s, want A in mock-greta and B in mock-hans", e.Value)
			}
			return
		case <-deadline:
			t.Fatal("no cast came")
		}
	}
}
