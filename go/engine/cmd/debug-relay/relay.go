package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// StatusBusy closes a tab that came while another holds the relay; the page
// shows it as busy and keeps trying.
const StatusBusy websocket.StatusCode = 4409

// localOrigins are the pages that may connect: any served from this machine.
var localOrigins = []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*", "[::1]", "[::1]:*"}

// Relay holds one playback-engine tab at a time and runs expressions in it
// for whoever asks over HTTP.
type Relay struct {
	timeout time.Duration

	mu  sync.Mutex
	tab *tab
}

// New is a relay that waits up to timeout for a tab to answer.
func New(timeout time.Duration) *Relay {
	return &Relay{timeout: timeout}
}

// tab is the connected page.
type tab struct {
	conn *websocket.Conn

	mu      sync.Mutex
	label   json.RawMessage
	next    int
	pending map[int]chan result
}

// result is a tab's answer to one expression: a value, or why there is none.
type result struct {
	Value json.RawMessage `json:"value"`
	Error string          `json:"error"`
}

// fromTab is everything a tab sends: its hello, and its results.
type fromTab struct {
	Type  string          `json:"type"`
	Label json.RawMessage `json:"label"`
	ID    int             `json:"id"`
	result
}

// toTab is an expression for the tab to run.
type toTab struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	Expr string `json:"expr"`
}

// errNoTab is asking a relay that holds no tab.
var errNoTab = errors.New("no tab connected: open a game on a dev server")

// Handler serves the page's connection and Claude's questions.
func (r *Relay) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /connect", r.connect)
	mux.HandleFunc("GET /tab", r.describeTab)
	mux.HandleFunc("POST /eval", func(w http.ResponseWriter, req *http.Request) {
		expr, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
		if err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		r.answer(w, req.Context(), string(expr))
	})
	mux.HandleFunc("GET /snapshot", func(w http.ResponseWriter, req *http.Request) {
		r.answer(w, req.Context(), "snapshot()")
	})
	return local(mux)
}

// local refuses what a page on another site could send, and an address that
// only resolves to this machine by a trick (DNS rebinding).
func local(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !isLocalHost(req.Host) {
			fail(w, http.StatusForbidden, fmt.Errorf("host %q is not this machine", req.Host))
			return
		}
		if origin := req.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !isLocalHost(u.Host) {
				fail(w, http.StatusForbidden, fmt.Errorf("origin %q is not this machine", origin))
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

// isLocalHost is whether a host, with or without its port, names this machine.
func isLocalHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	switch host {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	return false
}

// connect takes a tab, or tells it the relay is busy.
func (r *Relay) connect(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{OriginPatterns: localOrigins})
	if err != nil {
		return
	}
	t := &tab{conn: conn, pending: map[int]chan result{}}

	r.mu.Lock()
	if r.tab != nil {
		r.mu.Unlock()
		conn.Close(StatusBusy, "busy: another tab holds the relay")
		return
	}
	r.tab = t
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		if r.tab == t {
			r.tab = nil
		}
		r.mu.Unlock()
		t.drop()
		conn.CloseNow()
	}()

	// Only now is the tab the relay's: a busy relay accepts and closes, so
	// the page shows itself connected on this word alone.
	if err := wsjson.Write(req.Context(), conn, map[string]string{"type": "welcome"}); err != nil {
		return
	}

	conn.SetReadLimit(64 << 20)
	go keepAlive(req.Context(), conn)
	for {
		var msg fromTab
		if err := wsjson.Read(req.Context(), conn, &msg); err != nil {
			return
		}
		switch msg.Type {
		case "hello":
			t.mu.Lock()
			t.label = msg.Label
			t.mu.Unlock()
		case "result":
			t.deliver(msg.ID, msg.result)
		}
	}
}

// keepAlive pings the tab, so one that vanished without closing, as a
// sleeping laptop's does, frees the relay for the next.
func keepAlive(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ping, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(ping)
			cancel()
			if err != nil {
				conn.CloseNow()
				return
			}
		}
	}
}

// describeTab says which tab holds the relay, once it has said so itself.
func (r *Relay) describeTab(w http.ResponseWriter, _ *http.Request) {
	t := r.current()
	if t == nil {
		fail(w, http.StatusServiceUnavailable, errNoTab)
		return
	}
	t.mu.Lock()
	label := t.label
	t.mu.Unlock()
	if label == nil {
		fail(w, http.StatusServiceUnavailable, errors.New("a tab is connecting"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(label)
}

// answer runs an expression in the tab and writes its result.
func (r *Relay) answer(w http.ResponseWriter, ctx context.Context, expr string) {
	t := r.current()
	if t == nil {
		fail(w, http.StatusServiceUnavailable, errNoTab)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	res, err := t.eval(ctx, expr)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		fail(w, http.StatusGatewayTimeout, fmt.Errorf("the tab did not answer within %s", r.timeout))
	case err != nil:
		fail(w, http.StatusBadGateway, err)
	case res.Error != "":
		fail(w, http.StatusUnprocessableEntity, errors.New(res.Error))
	default:
		value := res.Value
		if value == nil {
			value = json.RawMessage("null")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(append(value, '\n'))
	}
}

// current is the tab that holds the relay, or nil.
func (r *Relay) current() *tab {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tab
}

// eval sends an expression and waits for its result.
func (t *tab) eval(ctx context.Context, expr string) (result, error) {
	reply := make(chan result, 1)
	t.mu.Lock()
	if t.pending == nil {
		t.mu.Unlock()
		return result{}, errors.New("the tab went away")
	}
	t.next++
	id := t.next
	t.pending[id] = reply
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
	}()

	if err := wsjson.Write(ctx, t.conn, toTab{Type: "eval", ID: id, Expr: expr}); err != nil {
		return result{}, err
	}
	select {
	case res, ok := <-reply:
		if !ok {
			return result{}, errors.New("the tab went away")
		}
		return res, nil
	case <-ctx.Done():
		return result{}, ctx.Err()
	}
}

// deliver hands a result to the expression waiting for it, if it still waits.
func (t *tab) deliver(id int, res result) {
	t.mu.Lock()
	reply := t.pending[id]
	delete(t.pending, id)
	t.mu.Unlock()
	if reply != nil {
		reply <- res
	}
}

// drop ends every wait, as the tab will answer none of them.
func (t *tab) drop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, reply := range t.pending {
		close(reply)
	}
	t.pending = nil
}

// fail answers with an error as JSON, the way every refusal is said.
func fail(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
