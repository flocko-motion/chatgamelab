package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"engine"
	"engine/api"
)

// The subtree must work identically wherever it is mounted, which is the whole
// reason the engine owns it: the player addresses it with relative URLs.
func TestHandlerWorksUnderAnyPrefix(t *testing.T) {
	for _, prefix := range []string{"", "/api/engine", "/deeply/nested/mount"} {
		t.Run("prefix="+prefix, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			api := api.New()
			s, err := engine.Launch(ctx, engine.SessionSpec{
				Genre: engine.GenreAdventure,
				ID:    "s1",
			})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			api.Register("s1", s)

			mux := http.NewServeMux()
			if prefix == "" {
				mux.Handle("/", api.Handler())
			} else {
				mux.Handle(prefix+"/", http.StripPrefix(prefix, api.Handler()))
			}
			srv := httptest.NewServer(mux)
			defer srv.Close()

			resp, err := http.Get(srv.URL + prefix + "/sessions/s1/graph")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("graph: status %d", resp.StatusCode)
			}
		})
	}
}

func TestInputAndStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := api.New()
	s, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	api.Register("s1", s)

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	streamCtx, stopStream := context.WithTimeout(ctx, 3*time.Second)
	defer stopStream()
	req, _ := http.NewRequestWithContext(streamCtx, http.MethodGet, srv.URL+"/sessions/s1/stream", nil)
	stream, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Body.Close()

	post, err := http.Post(srv.URL+"/sessions/s1/input", "application/json",
		strings.NewReader(`{"text":"I try to cross the bridge"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer post.Body.Close()
	if post.StatusCode != http.StatusAccepted {
		t.Fatalf("input: status %d", post.StatusCode)
	}

	buf := make([]byte, 512)
	n, err := stream.Body.Read(buf)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	if got := string(buf[:n]); !strings.Contains(got, "event: ") {
		t.Errorf("stream carried no SSE event, got %q", got)
	}
}

// An unsupported input kind is refused over HTTP too, rather than accepted and
// discarded.
func TestInputKindRefusedOverHTTP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := api.New()
	s, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "typed"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	api.Register("typed", s)

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/sessions/typed/input", "application/json",
		strings.NewReader(`{"audio":"c3Bva2Vu"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status %d, want 422", resp.StatusCode)
	}
}

func TestUnknownSessionIs404(t *testing.T) {
	srv := httptest.NewServer(api.New().Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sessions/nope/graph")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", resp.StatusCode)
	}
}

// newServer mounts the engine subtree and registers one launched session, which
// is what every test here needs before it can do anything.
func newServer(t *testing.T, id string, session *engine.Session) *httptest.Server {
	t.Helper()
	a := api.New()
	a.Register(id, session)
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	return srv
}
