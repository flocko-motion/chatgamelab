package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/openai"
)

func keyed(key string) KeyFunc {
	return func(context.Context) (string, error) { return key, nil }
}

// The platform checks a key before creating a session, and the player is
// told a dead key apart from a provider that is down.
func TestCheckOfferTellsRejectedFromUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	saved := openai.ModelsURL
	defer func() { openai.ModelsURL = saved }()

	offer := Offer{Platform: PlatformOpenAI, ID: "share", Keys: keyed("good")}

	openai.ModelsURL = srv.URL
	if err := CheckOffer(context.Background(), offer); err != nil {
		t.Errorf("a good key: %v", err)
	}
	offer.Keys = keyed("dead")
	if err := CheckOffer(context.Background(), offer); !errors.Is(err, ErrKeyRejected) {
		t.Errorf("a dead key: got %v, want ErrKeyRejected", err)
	}

	srv.Close()
	if err := CheckOffer(context.Background(), offer); !errors.Is(err, ErrUnreachable) {
		t.Errorf("a provider down: got %v, want ErrUnreachable", err)
	}
}

func TestCheckOfferPassesTheMock(t *testing.T) {
	if err := CheckOffer(context.Background(), MockOffer()); err != nil {
		t.Error(err)
	}
}

func TestCheckOfferWithoutAKey(t *testing.T) {
	err := CheckOffer(context.Background(), Offer{Platform: PlatformMistral, ID: "share"})
	if r := RefusalOf(err); r.Problem != ProblemNoKey {
		t.Errorf("got %v, want no_key", err)
	}
}

// A launch whose offers cannot run the genre is refused as that problem, so
// the playback-engine can say so.
func TestLaunchNamesTheGenreProblem(t *testing.T) {
	noLive := MockOffer()
	noLive.Caveats.Allow = map[Tier]Toggles{TierMedium: {Images: true}}
	noLive.Caveats.Default = Choice{Tier: TierMedium, Toggles: Toggles{Images: true}}
	_, err := Launch(context.Background(), SessionSpec{Genre: GenreNPCLive}, []Offer{noLive})
	if r := RefusalOf(err); r.Problem != ProblemGenre {
		t.Errorf("got %v (%s), want the genre problem", err, r.Problem)
	}
}

type rejectingImage struct{}

func (rejectingImage) Generate(context.Context, adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	return nil, adapters.Usage{}, fmt.Errorf("images api: 401 Unauthorized: %w", adapters.ErrKeyRejected)
}

// A key that fails during play is told to the player on its own stream,
// distinct from a lost connection.
func TestKeyFailingDuringPlayIsAProblem(t *testing.T) {
	s := launchOrFail(t, adventure(MockOffer()))
	s.slots.image.Set(rejectingImage{})
	_, _, _ = s.slots.image.Generate(context.Background(), adapters.ImageRequest{Prompt: "a bridge"})

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-s.Events():
			if ev.Stream != "problem" {
				continue
			}
			var r Refusal
			if err := json.Unmarshal([]byte(ev.Value), &r); err != nil || r.Problem != ProblemKeyRejected {
				t.Fatalf("problem event %q, want key_rejected", ev.Value)
			}
			return
		case <-deadline:
			t.Fatal("no problem event for a rejected key")
		}
	}
}

type brokenTool struct{}

func (brokenTool) Query(context.Context, string, string) (string, adapters.Usage, error) {
	return "", adapters.Usage{}, errors.New("responses api: 404 model not found")
}

// A failed call during play costs the turn, not the game: the player is told
// with the turn_failed code and the line they sent, and the session goes on.
func TestAFailedTurnGoesOn(t *testing.T) {
	s := launchOrFail(t, adventure(MockOffer()))
	waitStarted(t, s)
	s.slots.tool.Set(brokenTool{})
	if err := s.Say("I cross the bridge"); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for reported := false; !reported; {
		select {
		case ev := <-s.Events():
			if ev.Stream == "problem" {
				t.Fatalf("a failed turn ended the game: %s", ev.Value)
			}
			if ev.Stream != "error" {
				continue
			}
			var report ErrorReport
			if err := json.Unmarshal([]byte(ev.Value), &report); err != nil ||
				report != (ErrorReport{Code: ErrorTurnFailed, Text: "I cross the bridge"}) {
				t.Fatalf("error event %q, want turn_failed with the line", ev.Value)
			}
			reported = true
		case <-deadline:
			t.Fatal("no error event for a failed turn")
		}
	}
	if s.ctx.Err() != nil {
		t.Error("the session ended after a failed turn")
	}
}

// A preparation call the game cannot start without ends the game when it
// fails, and the player is told.
func TestAFailedPreparationEndsTheGame(t *testing.T) {
	// A key that cannot be had fails every call at once, the first being
	// preparation's.
	keyless := Offer{
		Platform: PlatformOpenAI, ID: "share",
		Keys: func(context.Context) (string, error) { return "", errors.New("the key is gone") },
		Caveats: Caveats{
			Allow:   map[Tier]Toggles{TierMedium: {Images: true}},
			Default: Choice{Tier: TierMedium, Toggles: Toggles{Images: true}},
		},
	}
	s := launchOrFail(t, adventure(keyless))

	deadline := time.After(2 * time.Second)
	for problem := false; !problem; {
		select {
		case ev := <-s.Events():
			if ev.Stream != "problem" {
				continue
			}
			var r Refusal
			if err := json.Unmarshal([]byte(ev.Value), &r); err != nil || r.Problem != ProblemFailed {
				t.Fatalf("problem event %q, want failed", ev.Value)
			}
			problem = true
		case <-deadline:
			t.Fatal("no problem event for a failed preparation call")
		}
	}
	select {
	case <-s.ctx.Done():
	case <-time.After(time.Second):
		t.Error("the session kept running after the game failed")
	}
}

// waitStarted waits for the gate to open.
func waitStarted(t *testing.T, s *Session) {
	t.Helper()
	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("the game never started")
	}
}
