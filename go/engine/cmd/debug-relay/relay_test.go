package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// fakeTab is a playback-engine page on a dev server, as the relay sees it.
type fakeTab struct {
	t        *testing.T
	conn     *websocket.Conn
	welcomed atomic.Bool
}

// openTab connects a tab with a label and answers every expression with answer.
func openTab(t *testing.T, relay *httptest.Server, label string, answer func(expr string) (any, string)) *fakeTab {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(relay.URL, "http")+"/connect", &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {"http://localhost:7100"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	tab := &fakeTab{t: t, conn: conn}
	// A busy relay may close the tab before its hello is out; closedWith tells.
	_ = wsjson.Write(ctx, conn, map[string]any{"type": "hello", "label": map[string]string{"game": label}})
	go func() {
		for {
			var msg struct {
				Type string `json:"type"`
				ID   int    `json:"id"`
				Expr string `json:"expr"`
			}
			if err := wsjson.Read(context.Background(), conn, &msg); err != nil {
				return
			}
			if msg.Type == "welcome" {
				tab.welcomed.Store(true)
				continue
			}
			value, failure := answer(msg.Expr)
			reply := map[string]any{"type": "result", "id": msg.ID}
			if failure != "" {
				reply["error"] = failure
			} else {
				reply["value"] = value
			}
			_ = wsjson.Write(context.Background(), conn, reply)
		}
	}()
	return tab
}

// closedWith waits for the relay to close the tab, and says with which code.
func (tab *fakeTab) closedWith() websocket.StatusCode {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		if _, _, err := tab.conn.Read(ctx); err != nil {
			return websocket.CloseStatus(err)
		}
	}
}

func echo(expr string) (any, string) { return map[string]string{"ran": expr}, "" }

func newRelay(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(New(2 * time.Second).Handler())
	t.Cleanup(server.Close)
	return server
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(body)
}

func post(t *testing.T, url, body string, header http.Header) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	for k, v := range header {
		req.Header[k] = v
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
}

// eventually retries a check until it holds, as a tab's hello arrives on its own time.
func eventually(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTheConnectedTabSaysWhoItIs(t *testing.T) {
	relay := newRelay(t)
	tab := openTab(t, relay, "Bridge Keeper", echo)

	eventually(t, tab.welcomed.Load)
	eventually(t, func() bool {
		code, body := get(t, relay.URL+"/tab")
		return code == 200 && strings.Contains(body, `"game":"Bridge Keeper"`)
	})
}

func TestAnExpressionRunsInTheTabAndItsResultComesBack(t *testing.T) {
	relay := newRelay(t)
	openTab(t, relay, "Bridge Keeper", echo)
	eventually(t, func() bool { code, _ := get(t, relay.URL+"/tab"); return code == 200 })

	code, body := post(t, relay.URL+"/eval", "playback.state.phases", nil)

	if code != 200 || strings.TrimSpace(body) != `{"ran":"playback.state.phases"}` {
		t.Fatalf("got %d %s", code, body)
	}
}

func TestASnapshotIsTheTabsSnapshot(t *testing.T) {
	relay := newRelay(t)
	openTab(t, relay, "Bridge Keeper", echo)
	eventually(t, func() bool { code, _ := get(t, relay.URL+"/tab"); return code == 200 })

	code, body := get(t, relay.URL+"/snapshot")

	if code != 200 || strings.TrimSpace(body) != `{"ran":"snapshot()"}` {
		t.Fatalf("got %d %s", code, body)
	}
}

func TestAnExpressionThatThrowsSaysWhy(t *testing.T) {
	relay := newRelay(t)
	openTab(t, relay, "Bridge Keeper", func(string) (any, string) { return nil, "ReferenceError: nope is not defined" })
	eventually(t, func() bool { code, _ := get(t, relay.URL+"/tab"); return code == 200 })

	code, body := post(t, relay.URL+"/eval", "nope", nil)

	var refused struct{ Error string }
	_ = json.Unmarshal([]byte(body), &refused)
	if code != http.StatusUnprocessableEntity || refused.Error != "ReferenceError: nope is not defined" {
		t.Fatalf("got %d %s", code, body)
	}
}

func TestWithNoTabThereIsNothingToAsk(t *testing.T) {
	relay := newRelay(t)

	tabCode, _ := get(t, relay.URL+"/tab")
	evalCode, _ := post(t, relay.URL+"/eval", "1", nil)

	if tabCode != http.StatusServiceUnavailable || evalCode != http.StatusServiceUnavailable {
		t.Fatalf("got /tab %d, /eval %d", tabCode, evalCode)
	}
}

func TestATabThatNeverAnswersTimesOut(t *testing.T) {
	server := httptest.NewServer(New(100 * time.Millisecond).Handler())
	t.Cleanup(server.Close)
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	openTab(t, server, "Bridge Keeper", func(string) (any, string) { <-block; return nil, "" })
	eventually(t, func() bool { code, _ := get(t, server.URL+"/tab"); return code == 200 })

	code, _ := post(t, server.URL+"/eval", "1", nil)

	if code != http.StatusGatewayTimeout {
		t.Fatalf("got %d", code)
	}
}

func TestASecondTabIsBusyAndGetsTheRelayOnceTheFirstGoes(t *testing.T) {
	relay := newRelay(t)
	first := openTab(t, relay, "First", echo)
	eventually(t, func() bool { _, body := get(t, relay.URL+"/tab"); return strings.Contains(body, "First") })

	second := openTab(t, relay, "Second", echo)
	if code := second.closedWith(); code != StatusBusy {
		t.Fatalf("second tab closed with %d, want busy", code)
	}
	if second.welcomed.Load() {
		t.Fatal("a busy tab was welcomed")
	}

	first.conn.Close(websocket.StatusNormalClosure, "")
	eventually(t, func() bool { code, _ := get(t, relay.URL+"/tab"); return code == http.StatusServiceUnavailable })
	openTab(t, relay, "Second", echo)
	eventually(t, func() bool { _, body := get(t, relay.URL+"/tab"); return strings.Contains(body, "Second") })
}

func TestATabFromAnotherSiteIsRefused(t *testing.T) {
	relay := newRelay(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(relay.URL, "http")+"/connect", &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {"https://example.com"}},
	})

	if err == nil {
		t.Fatal("a page from another site connected")
	}
}

func TestAnotherSiteCannotRunAnExpression(t *testing.T) {
	relay := newRelay(t)
	openTab(t, relay, "Bridge Keeper", echo)
	eventually(t, func() bool { code, _ := get(t, relay.URL+"/tab"); return code == 200 })

	code, _ := post(t, relay.URL+"/eval", "1", http.Header{"Origin": {"https://example.com"}})

	if code != http.StatusForbidden {
		t.Fatalf("got %d", code)
	}
}
