package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"engine"
	"engine/api"
)

func dialLive(t *testing.T, ctx context.Context, srv *httptest.Server, id string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/engine/sessions/" + id + "/live"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetReadLimit(-1)
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// readUntil reads a socket's text frames until one is a said line with want.
func readUntil(ctx context.Context, conn *websocket.Conn, want string) bool {
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return false
		}
		if typ != websocket.MessageText {
			continue
		}
		var ev engine.SessionEvent
		if json.Unmarshal(data, &ev) == nil && ev.Stream == "said" && ev.Value == want {
			return true
		}
	}
}

// Two pages on one session, as an old tab beside a reloaded one, each get
// every event: neither takes events from the other.
func TestTwoPagesOnASessionBothGetEverything(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "twice"}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	srv := newServer(t, "twice", session)

	old, reloaded := dialLive(t, ctx, srv, "twice"), dialLive(t, ctx, srv, "twice")
	// Both are subscribed once the session notes them; then the line goes in.
	for deadline := time.Now().Add(3 * time.Second); !attached(session, 2); {
		if time.Now().After(deadline) {
			t.Fatal("the pages never attached")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := session.Say("I knock on the door"); err != nil {
		t.Fatal(err)
	}
	for name, conn := range map[string]*websocket.Conn{"the old page": old, "the reloaded page": reloaded} {
		if !readUntil(ctx, conn, "I knock on the door") {
			t.Errorf("%s never got the player's line", name)
		}
	}
}

func attached(s *engine.Session, n int) bool {
	count := 0
	for _, e := range s.Journal() {
		if e.Source == engine.JournalClient && e.Kind == "attached" {
			count++
		}
	}
	return count >= n
}

// A session nobody has had a page open on for a while is unloaded, and the
// next request restores it from the store.
func TestAnIdleSessionIsUnloadedAndRestored(t *testing.T) {
	store := &engine.MemoryPersist{}
	var resumes atomic.Int32
	resume := func(ctx context.Context, id string) (*engine.Session, error) {
		resumes.Add(1)
		return engine.Resume(context.WithoutCancel(ctx), id, store, mockOffers(), nil)
	}
	a := api.New(api.Options{Resume: resume, UnloadAfter: 50 * time.Millisecond})
	defer a.Close()
	srv := httptest.NewServer(serve(a))
	defer srv.Close()

	first, err := engine.Launch(context.Background(), engine.SessionSpec{
		Genre: engine.GenreAdventure, ID: "idle", Persist: store,
	}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	a.Register("idle", first)

	for deadline := time.Now().Add(3 * time.Second); ; {
		resp, err := http.Get(srv.URL + "/api/engine/sessions/idle/state")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resumes.Load() > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the idle session was never unloaded and restored")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A session with a page open is not unloaded, however long it stays.
func TestASessionWithAPageOpenStays(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := &engine.MemoryPersist{}
	var resumes atomic.Int32
	resume := func(ctx context.Context, id string) (*engine.Session, error) {
		resumes.Add(1)
		return engine.Resume(context.WithoutCancel(ctx), id, store, mockOffers(), nil)
	}
	a := api.New(api.Options{Resume: resume, UnloadAfter: 50 * time.Millisecond})
	defer a.Close()
	srv := httptest.NewServer(serve(a))
	defer srv.Close()

	s, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreAdventure, ID: "open", Persist: store}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	a.Register("open", s)
	dialLive(t, ctx, srv, "open")
	for deadline := time.Now().Add(3 * time.Second); !attached(s, 1); {
		if time.Now().After(deadline) {
			t.Fatal("the page never attached")
		}
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	resp, err := http.Get(srv.URL + "/api/engine/sessions/open/state")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resumes.Load() != 0 {
		t.Error("a session with a page open was unloaded")
	}
}

// An observer's beat is a tuning knob: set on a dev server, refused elsewhere,
// and its state shows in the block's details.
func TestTheBeatIsSetOnDevServersOnly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session, err := engine.Launch(ctx, engine.SessionSpec{Genre: engine.GenreNPCLive, ID: "beat", Scenario: "A bridge."}, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	dev := newServer(t, "beat", session)
	resp := post(t, dev.URL+"/api/engine/sessions/beat/nodes/observer/beat", map[string]int{"ms": 2500})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("a dev server refused the beat: %d", resp.StatusCode)
	}
	if detail, _ := session.Inspect("observer"); detail.Observer == nil || detail.Observer.BeatMs != 2500 {
		t.Errorf("the observer's details do not show the new beat: %+v", detail.Observer)
	}

	a := api.New(api.Options{})
	a.Register("beat", session)
	plain := httptest.NewServer(serve(a))
	defer plain.Close()
	resp = post(t, plain.URL+"/api/engine/sessions/beat/nodes/observer/beat", map[string]int{"ms": 1000})
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("a server that is no dev server set the beat")
	}
}
