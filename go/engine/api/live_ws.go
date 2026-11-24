package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/coder/websocket"
	schemafapi "github.com/flocko-motion/schemaf/api"

	"engine"
)

// LiveEndpoint is the duplex WebSocket a voice genre needs.
//
// The browser sends binary frames of microphone audio and text frames of
// typed input, a JSON LiveInput; it receives the session's events back on the
// same connection, binary frames for audio and text frames of JSON
// SessionEvent for everything else. A rejected input comes back as an `error`
// event: a refused binary frame as the `voice_failed` report, so a page waiting
// for what was heard of a clip stops waiting. On a turn-based genre a binary
// frame is one whole clip, one utterance.
type LiveEndpoint struct {
	API *API
	// Guard, if set, is asked before each turn (turn true): a typed line, or
	// a clip on a genre that takes push-to-talk. While a live voice flows it
	// is asked at most every guardEvery (turn false). It says whether the
	// conversation may go on; a refusal goes out on the `problem` stream and
	// ends the socket.
	Guard func(ctx context.Context, turn bool) error
}

// guardEvery is how often a flowing voice conversation is re-checked.
const guardEvery = 30 * time.Second

func (LiveEndpoint) Method() string             { return "GET" }
func (LiveEndpoint) Path() string               { return "/api/engine/sessions/{id}/live" }
func (LiveEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e LiveEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	s, err := e.API.Session(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Same-origin only: the engine is mounted inside whatever serves the
		// playback-engine, so a cross-origin upgrade has no legitimate caller.
		InsecureSkipVerify: false,
	})
	if err != nil {
		// Accept has answered the failed upgrade itself.
		return nil
	}
	defer conn.CloseNow()
	conn.SetReadLimit(-1)

	// Every page attaching and leaving is noted: a reload shows as a pair.
	client := r.RemoteAddr
	s.Note(engine.JournalClient, "attached", "a page's socket from "+client)
	defer func() { s.Note(engine.JournalClient, "detached", "the socket from "+client) }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// This page's own copy of the session's events, ended with the socket.
	events, unsubscribe := s.Subscribe()
	defer unsubscribe()

	// A page that vanished without closing answers no ping, and its socket
	// is closed; it unsubscribes on the way out.
	go keepAlive(ctx, conn, cancel)

	// Outbound: the session's event stream, framed for the browser, and a
	// heartbeat the page listens for, as it cannot see the pings.
	go func() {
		// Ended either way, so the page reconnects and rebuilds when its
		// subscription was cut off for falling behind.
		defer cancel()
		beat := time.NewTicker(HeartbeatEvery)
		defer beat.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-beat.C:
				if err := wsJSON(ctx, conn, engine.SessionEvent{Stream: engine.StreamHeartbeat}); err != nil {
					return
				}
			case e, open := <-events:
				if !open {
					return
				}
				switch e.Stream {
				case "audio":
					if err := conn.Write(ctx, websocket.MessageBinary, []byte(e.Value)); err != nil {
						return
					}
				default:
					if err := wsJSON(ctx, conn, e); err != nil {
						return
					}
				}
			}
		}
	}()

	// Inbound: binary frames are microphone audio, text frames are typed input.
	// A live session takes both, and so does this endpoint. On a push-to-talk
	// genre each binary frame is a clip, a turn of its own.
	clips := slices.Contains(s.Topology().Inputs, "audio-push-to-talk")
	lastGuard := time.Now()
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return nil
		}
		if e.Guard != nil {
			turn := typ != websocket.MessageBinary || clips
			if turn || time.Since(lastGuard) >= guardEvery {
				lastGuard = time.Now()
				if err := e.Guard(ctx, turn); err != nil {
					refusal := engine.RefusalOf(err)
					blob, _ := json.Marshal(refusal)
					_ = wsJSON(ctx, conn, engine.SessionEvent{Stream: "problem", Value: string(blob)})
					return nil
				}
			}
		}

		var submitErr error
		if typ == websocket.MessageBinary {
			submitErr = s.Speak(data)
		} else {
			var req LiveInput
			if err := json.Unmarshal(data, &req); err != nil {
				submitErr = fmt.Errorf("malformed input: %w", err)
			} else {
				submitErr = s.Say(req.Text)
			}
		}

		if submitErr != nil {
			// Report and keep going: a client mistake is not a reason to end
			// someone's conversation.
			refused := submitErr.Error()
			if typ == websocket.MessageBinary {
				report, _ := json.Marshal(engine.ErrorReport{Code: engine.ErrorVoiceFailed})
				refused = string(report)
			}
			if err := wsJSON(ctx, conn, engine.SessionEvent{Stream: "error", Value: refused}); err != nil {
				return nil
			}
		}
	}
}

func wsJSON(ctx context.Context, conn *websocket.Conn, e engine.SessionEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, b)
}

// LiveInput is a text frame on the live WebSocket: typed input.
type LiveInput struct {
	Text string `json:"text"`
}

// How a page's socket is kept honest: a ping every PingEvery, answered within
// PingTimeout or the socket is closed; and a heartbeat event every
// HeartbeatEvery, which the page waits for, as a browser cannot see pings.
const (
	PingEvery      = 20 * time.Second
	PingTimeout    = 10 * time.Second
	HeartbeatEvery = 20 * time.Second
)

// keepAlive pings the page until the socket ends; an unanswered ping ends it.
func keepAlive(ctx context.Context, conn *websocket.Conn, end context.CancelFunc) {
	tick := time.NewTicker(PingEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			pingCtx, cancel := context.WithTimeout(ctx, PingTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				end()
				return
			}
		}
	}
}
