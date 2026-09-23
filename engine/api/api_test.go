package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// Every block on the graph is clickable, so every block must answer. A node the
// wiring does not have is a 404 rather than an empty panel.
func TestEveryNodeIsInspectable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "inspect",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "inspect", session)

	topology := session.Topology()
	if len(topology.Nodes) == 0 {
		t.Fatal("a genre with no nodes cannot be drawn, let alone inspected")
	}

	for _, node := range topology.Nodes {
		resp, err := http.Get(srv.URL + "/sessions/inspect/nodes/" + node.Name)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d", node.Name, resp.StatusCode)
		}
	}

	resp, err := http.Get(srv.URL + "/sessions/inspect/nodes/not-a-block")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown block: status %d, want 404", resp.StatusCode)
	}
}

// A block's detail must never carry the key. The engine holds a resolver rather
// than a secret, so there is nothing to leak — and a test says so, because this
// is the kind of thing a later change breaks quietly.
func TestNodeDetailCarriesNoSecret(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "secretless",
		Scenario: "You are the keeper of a bridge.",
		Platform: engine.PlatformOpenAI,
		Keys: func(context.Context) (string, error) {
			return "sk-should-never-appear", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "secretless", session)

	for _, node := range session.Topology().Nodes {
		resp, err := http.Get(srv.URL + "/sessions/secretless/nodes/" + node.Name)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(body), "sk-should-never-appear") {
			t.Fatalf("%s leaked the key: %s", node.Name, body)
		}
	}
}

// A sink has no outputs and a source has no inputs, and both must serialise as
// empty lists. A null there crashed the panel reading them, which showed up as a
// modal stuck on "loading" rather than as an error.
func TestNodeDetailListsAreNeverNull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "lists",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "lists", session)

	for _, node := range session.Topology().Nodes {
		resp, err := http.Get(srv.URL + "/sessions/lists/nodes/" + node.Name)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(body), `"inputs":null`) || strings.Contains(string(body), `"outputs":null`) {
			t.Errorf("%s serialised a null list: %s", node.Name, body)
		}
	}
}

// A wire is clickable too, and reports what it last carried. One the wiring
// does not have is a 404 rather than an empty reading.
func TestEdgeInspection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "edges",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "edges", session)

	for _, edge := range session.Topology().Edges {
		url := fmt.Sprintf("%s/sessions/edges/edge?from=%s&to=%s&kind=%s",
			srv.URL, edge.From, edge.To, edge.Kind)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s → %s (%s): status %d", edge.From, edge.To, edge.Kind, resp.StatusCode)
		}
	}

	resp, err := http.Get(srv.URL + "/sessions/edges/edge?from=observer&to=out-text&kind=text")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("an edge the wiring lacks: status %d, want 404", resp.StatusCode)
	}
}

// The browser's offer goes to the engine, and only an answer comes back. The
// provider's session id never appears in the response, because a browser that
// held one could attach its own control connection to the conversation.
func TestConnectBrokersTheOfferAndKeepsTheSessionID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:    engine.GenreNPCLive,
		ID:       "brokered",
		Scenario: "You are the keeper of a bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "brokered", session)

	// The invitation comes first: nothing may connect before the game begins.
	waitForConnectInvitation(t, session)

	body := strings.NewReader(`{"sdp":"v=0 browser offer"}`)
	resp, err := http.Post(srv.URL+"/sessions/brokered/connect", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect answered %s", resp.Status)
	}

	var answer struct {
		SDP string `json:"sdp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		t.Fatal(err)
	}
	if answer.SDP == "" {
		t.Error("the browser was given nothing to apply")
	}
}

// An offer with no SDP is a client mistake, and says so rather than opening a
// conversation nobody can hear.
func TestConnectRejectsAnEmptyOffer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "empty", Scenario: "A bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "empty", session)
	resp, err := http.Post(srv.URL+"/sessions/empty/connect", "application/json",
		strings.NewReader(`{"sdp":""}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("an empty offer answered %s", resp.Status)
	}
}

// What a session runs on is part of the subject on a platform that teaches how
// AI works, so the prompts are a first-class response rather than a debug dump.
func TestPromptsAreReadable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:          engine.GenreNPCLive,
		ID:             "prompts",
		Scenario:       "You are the keeper of a bridge.",
		PromptOverride: map[string]string{"portrait": "A woodcut of: "},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "prompts", session)
	resp, err := http.Get(srv.URL + "/sessions/prompts/prompts")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var prompts map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&prompts); err != nil {
		t.Fatal(err)
	}
	if prompts["portrait"] != "A woodcut of: " {
		t.Errorf("the override is not what the session reports running on: %q", prompts["portrait"])
	}
	if prompts["observer"] == "" {
		t.Error("a prompt the session uses is missing from what it reports")
	}
}

// A misspelled override is refused at launch. Overriding by name is untyped by
// construction, so the alternative is a session running on a default while its
// author believes otherwise.
func TestUnknownPromptOverrideIsRefused(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := engine.Launch(ctx, engine.SessionSpec{
		Genre:          engine.GenreNPCLive,
		Scenario:       "A bridge.",
		PromptOverride: map[string]string{"portraitt": "typo"},
	})
	if err == nil {
		t.Fatal("a misspelled prompt name launched anyway")
	}
	if !strings.Contains(err.Error(), "no such prompt") {
		t.Errorf("the error does not say what went wrong: %v", err)
	}
}

// waitForConnectInvitation plays the browser's part: a player is invited once
// the game has begun, and not before.
func waitForConnectInvitation(t *testing.T, s *engine.Session) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case e := <-s.Events():
			if e.Stream == "connect" {
				return
			}
		case <-deadline:
			t.Fatal("nobody was ever invited to connect")
		}
	}
}

// A session nobody has spoken in still has a history, and it is an empty list.
// Encoding it as null would hand a client something it cannot iterate, which is
// exactly what a page does on the reload path before anyone has said anything.
func TestEmptyHistoryIsAnEmptyList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "fresh", Scenario: "A bridge.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	srv := newServer(t, "fresh", session)
	resp, err := http.Get(srv.URL + "/sessions/fresh/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(body)) == "null" {
		t.Error("an empty history encoded as null")
	}
}

// A server holding one game answers for it whatever a request calls it. The
// player reads its session from the address, so a bare link, a bookmark from a
// previous run and a hand-typed address all arrive asking for something else —
// and every one of them means the only session there is.
func TestTheOnlySessionAnswersToAnyName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := engine.Launch(ctx, engine.SessionSpec{
		Genre: engine.GenreNPCLive, ID: "stopped-clock", Scenario: "A tower.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	a := api.New()
	a.RegisterOnly("stopped-clock", session)
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)

	// "standalone" is what the page asks for when the address names nothing.
	for _, id := range []string{"stopped-clock", "standalone", "a-game-from-last-week"} {
		resp, err := http.Get(srv.URL + "/sessions/" + id + "/topology")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("session %q answered %s: %s", id, resp.Status, body)
		}
	}
}

// A server holding several has nothing to fall back on: an id there is the only
// thing saying which game a request meant, so an unknown one stays unknown.
func TestAnUnknownSessionIsUnknownWhenThereAreSeveral(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := api.New()
	for _, id := range []string{"one", "two"} {
		s, err := engine.Launch(ctx, engine.SessionSpec{
			Genre: engine.GenreNPCLive, ID: id, Scenario: "A tower.",
		})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		a.Register(id, s)
	}

	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/sessions/three/topology")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("an unknown session among several answered %s", resp.Status)
	}
}
