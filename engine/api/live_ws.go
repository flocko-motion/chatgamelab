package api

import (
	"context"
	"net/http"

	"github.com/coder/websocket"

	"engine"
)

// SessionLive is the duplex endpoint a voice genre needs: the browser sends
// binary audio frames, and receives the session's events back on the same
// connection.
//
// The turn-based genres do not use this — they are served by SessionInput plus
// SessionStream. Which one a client opens is decided by the genre's declared
// interaction model, not by the client guessing.
func (a *API) SessionLive(w http.ResponseWriter, r *http.Request) {
	s, id, ok := a.lookup(r)
	if !ok {
		http.Error(w, "unknown session "+id, http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Same-origin only: the engine is mounted inside whatever serves the
		// player, so a cross-origin upgrade has no legitimate caller.
		InsecureSkipVerify: false,
	})
	if err != nil {
		return
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

	// Inbound: microphone frames.
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageBinary {
			continue
		}
		if err := s.Speak(data); err != nil {
			_ = wsJSON(ctx, conn, engine.Event{Stream: "error", Value: err.Error()})
			return
		}
	}
}

func wsJSON(ctx context.Context, conn *websocket.Conn, e engine.Event) error {
	b, err := jsonMarshal(e)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, b)
}
