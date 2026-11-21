package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
type LiveEndpoint struct{ API *API }

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

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Outbound: the session's event stream, framed for the browser.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e, open := <-s.Events():
				if !open {
					return
				}
				switch e.Stream {
				case "audio":
					if err := conn.Write(ctx, websocket.MessageBinary, []byte(e.Value)); err != nil {
						cancel()
						return
					}
				default:
					if err := wsJSON(ctx, conn, e); err != nil {
						cancel()
						return
					}
				}
			}
		}
	}()

	// Inbound: binary frames are microphone audio, text frames are typed input.
	// A live session takes both, and so does this endpoint.
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return nil
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
