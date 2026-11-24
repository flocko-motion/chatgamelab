package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"engine/internal/adapters"
)

// LiveSocketURL is GPT-Live's primary WebSocket, where the server sends and
// receives the audio itself; a variable so a test can point it at a fake.
var LiveSocketURL = "wss://api.openai.com/v1/live/sessions"

// A line is taken as said once the voice has been quiet for sayQuiet after
// it began: GPT-Live sends no event when its audio is done. sayLimit bounds
// the whole call, sayFinish the wait for the final usage after closing.
const (
	sayQuiet  = 1200 * time.Millisecond
	sayLimit  = 30 * time.Second
	sayFinish = 5 * time.Second
)

// Say has a GPT-Live voice say one line, over the primary WebSocket, and
// answers its audio as PCM at adapters.SayRate. A voice made from a
// description speaks only in GPT-Live, so its sample is spoken here.
// GPT-Live has no verbatim mode: the line goes in as an instruction to say
// it exactly, which it follows closely.
func (l *Live) Say(ctx context.Context, req adapters.SayRequest) ([]byte, adapters.Usage, error) {
	used := adapters.Usage{Model: l.model}
	key, err := l.keys(ctx)
	if err != nil {
		return nil, used, fmt.Errorf("resolve api key: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, sayLimit)
	defer cancel()

	ws, resp, err := websocket.Dial(ctx, LiveSocketURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + key}},
	})
	if err != nil {
		if resp != nil {
			return nil, used, adapters.StatusError("open gpt-live socket", resp, err.Error())
		}
		return nil, used, adapters.Transient("open gpt-live socket", err)
	}
	defer ws.CloseNow()
	ws.SetReadLimit(-1)

	// A reader of its own: a read whose context ends closes the socket, and
	// the quiet that ends the line is a timer, not a read's deadline.
	events := make(chan serverEvent, 64)
	go func() {
		defer close(events)
		for {
			_, raw, err := ws.Read(ctx)
			if err != nil {
				return
			}
			var ev serverEvent
			if json.Unmarshal(raw, &ev) != nil {
				continue
			}
			select {
			case events <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	send := func(msg map[string]any) error {
		b, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return ws.Write(ctx, websocket.MessageText, b)
	}

	voice := req.Voice
	if voice == "" {
		voice = DefaultVoice
	}
	if err := send(map[string]any{
		"type":     "session.start",
		"event_id": "say-start",
		"session": map[string]any{
			"model":        l.model,
			"instructions": req.Instructions,
			"audio": map[string]any{
				"format": map[string]any{"type": "audio/pcm", "rate": adapters.SayRate},
				"output": map[string]any{"voice": voiceField(voice)},
			},
		},
	}); err != nil {
		return nil, used, adapters.Transient("start gpt-live session", err)
	}

	var pcm []byte
	started, asked := false, false
	var quiet <-chan time.Time
loop:
	for {
		select {
		case <-ctx.Done():
			if len(pcm) > 0 {
				break loop
			}
			return nil, used, adapters.Transient("gpt-live said nothing", ctx.Err())
		case <-quiet:
			break loop
		case ev, open := <-events:
			if !open {
				if len(pcm) > 0 {
					break loop
				}
				return nil, used, adapters.Transient("gpt-live closed the socket", fmt.Errorf("before the line was said"))
			}
			switch ev.Type {
			case "session.started":
				if asked {
					continue
				}
				asked = true
				if err := send(map[string]any{
					"type":          "session.instructions.append",
					"event_id":      "say-line",
					"delegation_id": nil,
					"content":       "Immediately say the following exactly and in full, then stay silent:\n" + req.Line,
				}); err != nil {
					return nil, used, adapters.Transient("ask gpt-live to speak", err)
				}
			case "session.output_audio.delta":
				if raw, err := base64.StdEncoding.DecodeString(ev.Delta); err == nil {
					pcm = append(pcm, raw...)
					started = true
				}
				if started {
					quiet = time.After(sayQuiet)
				}
			case "session.usage.updated":
				// Cumulative: the latest is the total.
				used.AudioSeconds = ev.Usage.Seconds
			case "session.closed":
				used.AudioSeconds = max(used.AudioSeconds, ev.Usage.Seconds)
				break loop
			case "error":
				return nil, used, conversationError(ev.Error.Code, ev.Error.Message)
			}
		}
	}

	// Closing waits for the final usage, so the call is billed in full.
	if send(map[string]any{"type": "session.close"}) == nil {
		finish := time.After(sayFinish)
	closing:
		for {
			select {
			case <-finish:
				break closing
			case <-ctx.Done():
				break closing
			case ev, open := <-events:
				if !open {
					break closing
				}
				if ev.Type == "session.usage.updated" || ev.Type == "session.closed" {
					used.AudioSeconds = max(used.AudioSeconds, ev.Usage.Seconds)
				}
				if ev.Type == "session.closed" {
					break closing
				}
			}
		}
	}
	_ = ws.Close(websocket.StatusNormalClosure, "")
	return pcm, used, nil
}
