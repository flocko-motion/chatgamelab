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
	"engine/internal/ports"
)

func keyed(key string) KeyFunc {
	return func(context.Context) (string, error) { return key, nil }
}

// The platform checks a key before creating a session and when one is
// saved, and the player is told a dead key, an account with no credit and a
// provider that is down apart. The check is one small billed call, so it
// finds an empty account too.
func TestCheckOfferTellsTheKeysProblems(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&call)
		asked = append(asked, call.Model+" "+call.Input)
		switch r.Header.Get("Authorization") {
		case "Bearer good":
			_, _ = w.Write([]byte(`{"output":[{"content":[{"text":"2"}]}]}`))
		case "Bearer empty":
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"insufficient_quota"}}`))
		case "Bearer busy":
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"rate_limit_exceeded"}}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	saved := openai.ResponsesURL
	defer func() { openai.ResponsesURL = saved }()
	openai.ResponsesURL = srv.URL

	check := func(key string) error {
		return CheckOffer(context.Background(), Offer{Platform: PlatformOpenAI, ID: "share", Keys: keyed(key)})
	}
	if err := check("good"); err != nil {
		t.Errorf("a good key: %v", err)
	}
	if len(asked) != 1 || asked[0] != lowestTooling(t, PlatformOpenAI)+" "+adapters.ProbePrompt {
		t.Errorf("the check asked %q, want the probe on the lowest tier's tooling model", asked)
	}
	for key, want := range map[string]error{
		"dead":  ErrKeyRejected,
		"empty": ErrNoCredit,
		"busy":  ErrUnreachable,
	} {
		if err := check(key); !errors.Is(err, want) {
			t.Errorf("key %q: got %v, want %v", key, err, want)
		}
	}

	srv.Close()
	if err := check("good"); !errors.Is(err, ErrUnreachable) {
		t.Errorf("a provider down: got %v, want ErrUnreachable", err)
	}
}

func lowestTooling(t *testing.T, platform Platform) string {
	t.Helper()
	entry, _ := catalogueEntry(platform)
	models, err := entry.Models(TierLow)
	if err != nil {
		t.Fatal(err)
	}
	return models[adapters.ModelTooling].Name
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

// A failure reaching the problem sink is told by its reason, so the player
// hears that the key was refused or the provider is away, once, and the
// session ends.
func TestAProblemNamesTheFailuresReason(t *testing.T) {
	for reason, want := range map[string]Problem{
		ports.ReasonKeyRejected: ProblemKeyRejected,
		ports.ReasonUnreachable: ProblemUnreachable,
		ports.ReasonBadReply:    ProblemFailed,
	} {
		t.Run(reason, func(t *testing.T) {
			s := launchOrFail(t, adventure(MockOffer()))
			waitStarted(t, s)
			failure := `{"node":"image-style","reason":"` + reason + `","message":"the provider said no"}`
			s.failed(failure)
			s.failed(failure)

			var problems []Refusal
			deadline := time.After(time.Second)
		read:
			for {
				select {
				case ev := <-s.Events():
					if ev.Stream != "problem" {
						continue
					}
					var r Refusal
					if err := json.Unmarshal([]byte(ev.Value), &r); err != nil {
						t.Fatal(err)
					}
					problems = append(problems, r)
				case <-deadline:
					break read
				}
			}
			if len(problems) != 1 || problems[0].Problem != want || problems[0].Message != "image-style: the provider said no" {
				t.Errorf("problems %+v, want one %s naming the block", problems, want)
			}
			if s.ctx.Err() == nil {
				t.Error("the session kept running")
			}
		})
	}
}
