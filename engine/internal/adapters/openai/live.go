package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"engine/internal/adapters"
)

type Live struct {
	keys  adapters.KeyFunc
	model string
}

func NewLive(keys adapters.KeyFunc, model string) *Live { return &Live{keys: keys, model: model} }

func (l *Live) Open(ctx context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	// Resolved here rather than held, so a key rotated between sessions is the
	// one this connection uses.
	key, err := l.keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve api key: %w", err)
	}

	url := fmt.Sprintf("%s?model=%s", realtimeURL, l.model)
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + key}},
	})
	if err != nil {
		return nil, fmt.Errorf("dial realtime: %w", err)
	}
	// No read limit: audio deltas are large and continuous.
	conn.SetReadLimit(-1)

	c := &liveConn{
		ws:     conn,
		events: make(chan adapters.LiveEvent, 256),
		done:   make(chan struct{}),
	}

	// The player's own speech is deliberately not transcribed: nobody re-reads
	// what they just said, and an ASR transcript that diverges from what the
	// conversation model acted on is a poor record to keep.
	if err := c.write(ctx, map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"instructions": cfg.Instructions,
			"voice":        cfg.Voice,
			"modalities":   []string{"audio", "text"},
		},
	}); err != nil {
		conn.CloseNow()
		return nil, err
	}

	go c.read(ctx)
	go c.meter(ctx, l.model)
	return c, nil
}

type liveConn struct {
	ws     *websocket.Conn
	events chan adapters.LiveEvent
	done   chan struct{}

	mu     sync.Mutex
	closed bool
}

func (c *liveConn) write(ctx context.Context, msg map[string]any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.Write(ctx, websocket.MessageText, b)
}

func (c *liveConn) Send(audio []byte) error {
	return c.write(context.Background(), map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(audio),
	})
}

// Instruct replaces the standing instructions mid-conversation. It steers what
// the character says next.
func (c *liveConn) Instruct(text string) error {
	return c.write(context.Background(), map[string]any{
		"type":    "session.update",
		"session": map[string]any{"instructions": text},
	})
}

func (c *liveConn) Events() <-chan adapters.LiveEvent { return c.events }

func (c *liveConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)
	return c.ws.Close(websocket.StatusNormalClosure, "")
}

// meter reports audio time as it accrues. A realtime session is billed by the
// minute, so waiting until the call ends would mean showing nothing for the
// whole conversation — which is exactly when someone wants to see the cost.
func (c *liveConn) meter(ctx context.Context, model string) {
	const tick = time.Second
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	elapsed := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
			elapsed += tick.Seconds()
			c.emit(adapters.LiveEvent{
				Kind:  adapters.EventUsage,
				Usage: adapters.Usage{Model: model, AudioSeconds: elapsed},
			})
		}
	}
}

type serverEvent struct {
	Type       string `json:"type"`
	Delta      string `json:"delta"`
	Transcript string `json:"transcript"`
	Error      struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *liveConn) read(ctx context.Context) {
	defer close(c.events)
	for {
		_, data, err := c.ws.Read(ctx)
		if err != nil {
			select {
			case <-c.done:
			default:
				c.emit(adapters.LiveEvent{Kind: adapters.EventError, Err: err})
			}
			return
		}

		var ev serverEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "response.output_audio.delta":
			raw, err := base64.StdEncoding.DecodeString(ev.Delta)
			if err != nil {
				continue
			}
			c.emit(adapters.LiveEvent{Kind: adapters.EventAudio, Audio: raw})

		case "response.output_audio_transcript.delta":
			// The model's own transcript of its own speech: exact, not an ASR
			// guess, which is why this is what the observer judges.
			c.emit(adapters.LiveEvent{Kind: adapters.EventText, Text: ev.Delta})

		case "response.done":
			c.emit(adapters.LiveEvent{Kind: adapters.EventTurnComplete})

		case "error":
			c.emit(adapters.LiveEvent{
				Kind: adapters.EventError,
				Err:  fmt.Errorf("realtime: %s", ev.Error.Message),
			})
		}
	}
}

func (c *liveConn) emit(e adapters.LiveEvent) {
	select {
	case c.events <- e:
	case <-c.done:
	}
}
