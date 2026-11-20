package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"engine"
	"engine/api"
)

// creator launches on the mock and counts its launches, as a platform would
// create sessions from games.
type creator struct {
	store    *engine.MemoryPersist
	launches atomic.Int32
	refuse   error
}

func (c *creator) create(ctx context.Context, req api.CreateRequest) (*engine.Session, error) {
	if c.refuse != nil {
		return nil, c.refuse
	}
	n := c.launches.Add(1)
	return engine.Launch(context.WithoutCancel(ctx), engine.SessionSpec{
		Genre:    engine.GenreAdventure,
		ID:       fmt.Sprintf("%s-%d", req.Game, n),
		Language: req.Language,
		Persist:  c.store,
	}, mockOffers())
}

func (c *creator) resume(ctx context.Context, id string) (*engine.Session, error) {
	return engine.Resume(context.WithoutCancel(ctx), id, c.store, mockOffers())
}

func (c *creator) server(t *testing.T) (*api.API, *httptest.Server) {
	t.Helper()
	a := api.New(api.Options{Dev: true, Create: c.create, Resume: c.resume})
	srv := httptest.NewServer(serve(a))
	t.Cleanup(srv.Close)
	return a, srv
}

func post(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	blob, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(blob))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func created(t *testing.T, resp *http.Response) string {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create answered %s", resp.Status)
	}
	var out api.CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.ID
}

// A repeated `new` within a short time, such as a reload before the URL was
// replaced, answers the session the first one created rather than a second.
func TestOneTimeKeyAnswersTheSameSession(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}}
	_, srv := c.server(t)

	first := created(t, post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k1"}))
	again := created(t, post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k1"}))
	other := created(t, post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k2"}))

	if first != again {
		t.Errorf("the same key created %q and %q", first, again)
	}
	if other == first {
		t.Error("a new key answered the old session")
	}
	if n := c.launches.Load(); n != 2 {
		t.Errorf("%d launches, want 2", n)
	}

	resp, err := http.Get(srv.URL + "/api/engine/sessions/" + first + "/topology")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("the created session answered %s", resp.Status)
	}
}

// A one-time key answers only the caller who made it: another caller who
// knows it creates nothing and learns no session ID.
func TestOneTimeKeyBelongsToItsCaller(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}}
	a := api.New(api.Options{
		Dev:    true,
		Create: c.create,
		Caller: func(ctx context.Context) string { return schemafapi.Header(ctx, "X-Caller") },
	})
	srv := httptest.NewServer(serve(a))
	t.Cleanup(srv.Close)

	ask := func(caller string) *http.Response {
		blob, _ := json.Marshal(api.CreateRequest{Game: "bridge", Once: "k1"})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/engine/sessions", bytes.NewReader(blob))
		req.Header.Set("X-Caller", caller)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}

	first := created(t, ask("alice"))
	if again := created(t, ask("alice")); again != first {
		t.Errorf("the same caller got %q, then %q", first, again)
	}

	resp := ask("mallory")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("another caller's repeat answered %s, want 403", resp.Status)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if _, leaked := body["id"]; leaked {
		t.Errorf("another caller learnt a session ID: %v", body)
	}
	if body["code"] != string(engine.ProblemDenied) {
		t.Errorf("code %v, want %q", body["code"], engine.ProblemDenied)
	}
	if n := c.launches.Load(); n != 1 {
		t.Errorf("%d launches, want 1", n)
	}
}

func TestCreateNeedsAGameOrShareAndAKey(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}}
	_, srv := c.server(t)
	for _, req := range []api.CreateRequest{
		{Game: "bridge"},
		{Once: "k"},
		{Game: "bridge", Share: "tok", Once: "k"},
	} {
		if resp := post(t, srv.URL+"/api/engine/sessions", req); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%+v answered %s", req, resp.Status)
		}
	}
}

// Every launch error reaches the playback-engine with its problem.
func TestCreateRefusalReachesThePlaybackEngine(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}, refuse: engine.Refuse(engine.ProblemNoPlays, "the share has no plays left")}
	_, srv := c.server(t)

	resp := post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Share: "tok", Once: "k"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status %s", resp.Status)
	}
	if got := errorOf(t, resp); got.Code != string(engine.ProblemNoPlays) || got.Error != "the share has no plays left" {
		t.Errorf("refusal = %+v", got)
	}

	// A failure is not remembered: the same key tries again.
	c.refuse = nil
	created(t, post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Share: "tok", Once: "k"}))
}

// A session that is stored but not in memory, as after a restart, is
// restored by the first request that names it.
func TestLazyRestore(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}}
	a, srv := c.server(t)
	id := created(t, post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k", Language: "de"}))

	// What a restart leaves: the store, and nothing in memory.
	a.Forget(id)

	resp, err := http.Get(srv.URL + "/api/engine/sessions/" + id + "/topology")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var topology struct {
		Language string `json:"language"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&topology); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("restored session answered %s (%v)", resp.Status, err)
	}
	if topology.Language != "de" {
		t.Errorf("language = %q, want the stored one", topology.Language)
	}
}

// A session that can no longer be resumed says so, distinct from one that
// does not exist, so the playback-engine offers to start anew.
func TestStaleSessionIsGone(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}}
	a, srv := c.server(t)
	id := created(t, post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k"}))
	a.Forget(id)

	stored, err := c.store.LoadState(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	stored.Version++
	_ = c.store.SaveState(context.Background(), id, stored)

	resp, err := http.Get(srv.URL + "/api/engine/sessions/" + id + "/topology")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := errorOf(t, resp); resp.StatusCode != http.StatusGone || got.Code != string(engine.ProblemStale) {
		t.Errorf("answered %s with %+v, want gone and stale", resp.Status, got)
	}
}

// errorBody is schemaf's error response, which carries a refusal's problem as its code.
type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func errorOf(t *testing.T, resp *http.Response) errorBody {
	t.Helper()
	var body errorBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("error body: %v", err)
	}
	return body
}

// A refusal answers in schemaf's error format, literally, with its problem as
// the code: the playback-engine reads nothing else.
func TestARefusalAnswersErrorAndCode(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}, refuse: engine.Refuse(engine.ProblemDenied, "not yours")}
	_, srv := c.server(t)

	resp := post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status %s, want 403", resp.Status)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"error": "not yours", "code": "denied"}
	if fmt.Sprint(body) != fmt.Sprint(want) {
		t.Errorf("body = %v, want %v", body, want)
	}
}

// A refusal for a failure keeps its code while schemaf withholds the message.
func TestAFailureKeepsItsCode(t *testing.T) {
	c := &creator{store: &engine.MemoryPersist{}, refuse: errors.New("the database is on fire")}
	_, srv := c.server(t)

	resp := post(t, srv.URL+"/api/engine/sessions", api.CreateRequest{Game: "bridge", Once: "k"})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status %s, want 500", resp.Status)
	}
	if got := errorOf(t, resp); got.Code != string(engine.ProblemFailed) || strings.Contains(got.Error, "fire") {
		t.Errorf("body = %+v, want code failed and no message", got)
	}
}
