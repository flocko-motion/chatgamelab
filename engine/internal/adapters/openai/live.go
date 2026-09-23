package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"engine/internal/adapters"
)

// Live opens GPT-Live conversations.
//
// The audio runs between the player's browser and OpenAI over WebRTC; this
// adapter creates the session, hands back the SDP answer, and then attaches a
// second connection — a sideband — that carries transcripts, usage and control.
// The API key is therefore only ever on this side, and OpenAI's session id
// never reaches the browser.
type Live struct {
	keys  adapters.KeyFunc
	model string
	// sessions is where conversations are created. A field rather than the
	// constant directly, so a test can point it at a server that answers the
	// way the real one does.
	sessions string
	client   *http.Client
}

func NewLive(keys adapters.KeyFunc, model string) *Live {
	return &Live{
		keys:     keys,
		model:    model,
		sessions: liveSessionsURL,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// builtinVoices is what GPT-Live will accept. A voice cannot change once a
// session exists, so a bad name is worth catching before one is created rather
// than as a rejected handshake.
//
// TODO: a custom voice trained from a recording is referenced as an object
// carrying its id, which this adapter does not yet reach for.
var builtinVoices = map[string]bool{
	"alloy": true, "ash": true, "ballad": true, "beacon": true, "bossa": true,
	"cedar": true, "cinder": true, "coral": true, "delta": true, "echo": true,
	"gleam": true, "marin": true, "meridian": true, "quartz": true, "ripple": true,
	"sage": true, "shimmer": true, "stone": true, "tempo": true, "verse": true,
	"vesper": true, "willow": true,
}

// DefaultVoice is GPT-Live's own default, used when a spec names none.
const DefaultVoice = "marin"

// ValidVoice reports whether a name is one GPT-Live knows.
func ValidVoice(name string) bool { return builtinVoices[name] }

func (l *Live) Open(ctx context.Context, cfg adapters.LiveConfig) (adapters.LiveConn, error) {
	if cfg.Offer == "" {
		return nil, fmt.Errorf("gpt-live: no SDP offer; the browser opens the audio connection")
	}
	voice := cfg.Voice
	if voice == "" {
		voice = DefaultVoice
	}
	if !ValidVoice(voice) {
		return nil, fmt.Errorf("gpt-live: unknown voice %q", voice)
	}

	key, err := l.keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve api key: %w", err)
	}

	id, answer, err := l.create(ctx, key, cfg, voice)
	if err != nil {
		return nil, err
	}

	conn, err := l.attach(ctx, key, id, l.model)
	if err != nil {
		return nil, fmt.Errorf("attach sideband to %s: %w", id, err)
	}
	conn.answer = answer

	go conn.read(ctx)
	return conn, nil
}

// session is the configuration GPT-Live is created with. Everything here is
// immutable for the session's lifetime — model, voice and delegation mode
// included — which is why a change of any of them means a new conversation.
type session struct {
	Model string `json:"model"`
	// Instructions is the character: the game designer's scenario, and the
	// conversation policies the model's own prompt guide asks for.
	Instructions string `json:"instructions"`
	// Input seeds the conversation. The guardrail rides in here as a developer
	// message rather than being concatenated into Instructions, so a designer's
	// text can never occupy the field the platform's constraint holds.
	//
	// TODO: this field also takes up to 128 prior messages and 8,192 tokens of
	// transcript, which is how a dropped conversation would resume.
	Input []message `json:"input,omitempty"`
	Audio audioCfg  `json:"audio"`
	// Store keeps the recording, which is what makes a paused conversation
	// resumable: a fork continues from it. The platform already persists a
	// game's history, so this is the same posture in a second place — with the
	// difference, which belongs in the disclaimer, that this copy expires on
	// OpenAI's schedule rather than on request.
	Store bool `json:"store"`
}

type message struct {
	Type    string        `json:"type"`
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type audioCfg struct {
	Output audioOutCfg `json:"output"`
}

type audioOutCfg struct {
	Voice string `json:"voice"`
}

// create exchanges the browser's offer for an answer, which is the act that
// brings the session into being. It bills fifteen seconds up front, credited
// back once the conversation runs — so it happens when a player has arrived
// rather than when a session is launched.
// create brings a conversation into being, continuing an earlier one where it
// can.
//
// A fork is the better resume — the model picks up its own recording — but it
// needs that recording to have finalised, which only a graceful close produces.
// A dropped connection, a closed laptop or a killed process leave none, and
// those are exactly the moments somebody wants to carry on from. So a fork that
// cannot be honoured falls back to seeding a fresh session with what was said,
// rather than failing the resume outright.
func (l *Live) create(ctx context.Context, key string, cfg adapters.LiveConfig, voice string) (id, answer string, err error) {
	if cfg.ResumeFrom != "" {
		id, answer, err = l.post(ctx, key, l.sessions+"/"+cfg.ResumeFrom+"/fork", map[string]any{
			"transport": map[string]any{"type": "webrtc", "sdp": cfg.Offer},
			// A fork inherits model, instructions and history; only a few
			// fields may be overridden, so this is deliberately almost empty.
			"session": map[string]any{"store": true},
		})
		if err == nil {
			return id, answer, nil
		}
		// Falling back rather than reporting: no recording is an ordinary
		// outcome, not a fault, and the conversation carries on either way.
	}

	return l.post(ctx, key, l.sessions, map[string]any{
		"transport": map[string]any{"type": "webrtc", "sdp": cfg.Offer},
		"session": session{
			Model:        l.model,
			Instructions: cfg.Scenario,
			Input:        append(guardrailMessage(cfg.Guardrail), history(cfg.History)...),
			Audio:        audioCfg{Output: audioOutCfg{Voice: voice}},
			Store:        true,
		},
	})
}

func (l *Live) post(ctx context.Context, key, url string, payload map[string]any) (id, answer string, err error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		detail, _ := readError(resp)
		return "", "", fmt.Errorf("create live session: %s%s", resp.Status, detail)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	return readCreated(raw)
}

// createdSession is what a successful creation answers with. Both fields are
// nested, which is worth a type of its own: reading them from the wrong level
// costs a session that has been created and billed before anything notices.
type createdSession struct {
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
	Transport struct {
		SDP string `json:"sdp"`
	} `json:"transport"`
}

func readCreated(raw []byte) (id, answer string, err error) {
	var out createdSession
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", fmt.Errorf("create live session: %w (body: %s)", err, excerpt(raw))
	}
	if out.Session.ID == "" || out.Transport.SDP == "" {
		// The session exists by now and is being billed, so the body goes into
		// the error: a shape that changed under us is worth seeing once rather
		// than guessing at twice.
		return "", "", fmt.Errorf("create live session: response carried no %s (body: %s)",
			missing(out.Session.ID, out.Transport.SDP), excerpt(raw))
	}
	return out.Session.ID, out.Transport.SDP, nil
}

// guardrailMessage puts the platform's constraint in as a developer message,
// which is where the API documents trusted application instructions belong.
func guardrailMessage(guardrail string) []message {
	if guardrail == "" {
		return nil
	}
	return []message{{
		Type:    "message",
		Role:    "developer",
		Content: []contentPart{{Type: "input_text", Text: guardrail}},
	}}
}

// excerpt trims a response body down to something an error can carry. An SDP
// answer is kilobytes of candidates, and none of it helps read a mistake.
func excerpt(raw []byte) string {
	const limit = 300
	if len(raw) <= limit {
		return string(raw)
	}
	return string(raw[:limit]) + "…"
}

// history turns what was said into the startup messages that restore it. This
// is the path taken when the provider has no finalised recording to fork — a
// dropped connection leaves none, and that is exactly when somebody most wants
// to carry on.
//
// The provider accepts at most 128 messages and 8,192 tokens, so the oldest are
// dropped first: a character remembering the last few exchanges and forgetting
// the opening is better than one refused a resume for talking too long.
func history(said []adapters.Utterance) []message {
	const most = 100

	if len(said) > most {
		said = said[len(said)-most:]
	}

	out := make([]message, 0, len(said))
	for _, one := range said {
		if one.Text == "" {
			continue
		}
		role, part := "assistant", "output_text"
		if one.Speaker == adapters.SpeakerPlayer {
			role, part = "user", "input_text"
		}
		out = append(out, message{
			Type:    "message",
			Role:    role,
			Content: []contentPart{{Type: part, Text: one.Text}},
		})
	}
	return out
}

func missing(id, sdp string) string {
	switch {
	case id == "" && sdp == "":
		return "session id or SDP answer"
	case id == "":
		return "session id"
	default:
		return "SDP answer"
	}
}

func (l *Live) attach(ctx context.Context, key, id, model string) (*liveConn, error) {
	// The attach endpoint is the sessions URL over a websocket scheme, which
	// keeps one constant rather than two that can drift apart.
	url := "wss" + strings.TrimPrefix(l.sessions, "https") + "/" + id + "/attach"
	ws, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + key}},
	})
	if err != nil {
		return nil, err
	}
	// No read limit: reflected audio arrives on this socket and is large.
	ws.SetReadLimit(-1)

	return &liveConn{
		ws:        ws,
		id:        id,
		modelID:   model,
		events:    make(chan adapters.LiveEvent, 256),
		done:      make(chan struct{}),
		finalised: make(chan struct{}),
		nextID:    1,
		pending:   map[string]struct{}{},
	}, nil
}

type liveConn struct {
	ws      *websocket.Conn
	id      string
	modelID string
	answer  string
	events  chan adapters.LiveEvent
	done    chan struct{}
	// finalised closes when the provider confirms the session has ended, which
	// is also when its recording is saved. Resuming by fork needs that
	// recording, so a close that did not wait for this leaves nothing to
	// continue from.
	finalised chan struct{}

	mu     sync.Mutex
	closed bool
	nextID int
	// pending tracks instruction appends awaiting acknowledgement, so a
	// rejected steer is reported rather than assumed to have landed.
	pending map[string]struct{}
}

func (c *liveConn) ID() string     { return c.id }
func (c *liveConn) Answer() string { return c.answer }

func (c *liveConn) Events() <-chan adapters.LiveEvent { return c.events }

func (c *liveConn) write(ctx context.Context, msg map[string]any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("gpt-live: session %s is closed", c.id)
	}
	return c.ws.Write(ctx, websocket.MessageText, b)
}

// Instruct appends to the standing instructions. Appending rather than
// replacing is what lets the guardrail be re-injected mid-conversation without
// discarding the character it was meant to constrain.
func (c *liveConn) Instruct(ctx context.Context, text string) error {
	c.mu.Lock()
	id := fmt.Sprintf("steer-%d", c.nextID)
	c.nextID++
	c.pending[id] = struct{}{}
	c.mu.Unlock()

	return c.write(ctx, map[string]any{
		"type":     "session.instructions.append",
		"event_id": id,
		// Null addresses the session as a whole rather than one delegated task.
		"delegation_id": nil,
		"content":       text,
	})
}

// Close ends the conversation and waits for the provider to say it has
// finished.
//
// The wait is the point. Asking to close and hanging up immediately leaves the
// session unfinalised, which costs the final usage figure and — because a fork
// continues a completed recording — the ability to resume at all. The symptom
// is a later resume answered with "the stored session was not found", long
// after the mistake.
func (c *liveConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	closing, cancel := context.WithTimeout(context.Background(), finalisationWait)
	defer cancel()

	b, _ := json.Marshal(map[string]any{"type": "session.close"})
	if err := c.ws.Write(closing, websocket.MessageText, b); err == nil {
		// Saving the recording takes its own time, so this waits for the
		// provider's word rather than for a guess at how long that is.
		select {
		case <-c.finalised:
		case <-closing.Done():
		}
	}

	close(c.done)
	return c.ws.Close(websocket.StatusNormalClosure, "")
}

// finalisationWait bounds the wait for a session to finish. Long enough for a
// recording to be saved, short enough that an unresponsive provider does not
// hold a paused game open.
const finalisationWait = 8 * time.Second

// serverEvent covers the sideband events this adapter acts on. Fields are
// shared across event types, which the wire format allows because a given type
// only ever populates its own.
type serverEvent struct {
	Type          string `json:"type"`
	Delta         string `json:"delta"`
	Audio         string `json:"audio"`
	StartMs       int64  `json:"start_ms"`
	EndMs         int64  `json:"end_ms"`
	ClientEventID string `json:"client_event_id"`
	Reason        string `json:"reason"`
	Usage         struct {
		Seconds float64 `json:"seconds"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *liveConn) read(ctx context.Context) {
	defer func() {
		// However the read ends, nothing more is coming: a Close still waiting
		// for finalisation should stop waiting rather than sit out its timeout.
		c.finish()
		close(c.events)
	}()

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
		case "session.started":
			c.emit(adapters.LiveEvent{Kind: adapters.EventStarted})

		case "session.output_transcript.delta":
			c.emit(adapters.LiveEvent{
				Kind:    adapters.EventText,
				Speaker: adapters.SpeakerCharacter,
				Text:    ev.Delta,
				StartMs: ev.StartMs,
				EndMs:   ev.EndMs,
			})

		case "session.input_transcript.delta":
			// The player's own words. Received because they arrive whether or
			// not they are wanted; what becomes of them is the wiring's call.
			c.emit(adapters.LiveEvent{
				Kind:    adapters.EventText,
				Speaker: adapters.SpeakerPlayer,
				Text:    ev.Delta,
				StartMs: ev.StartMs,
				EndMs:   ev.EndMs,
			})

		case "session.output_audio.delta":
			if raw, err := base64.StdEncoding.DecodeString(ev.Delta); err == nil {
				c.emit(adapters.LiveEvent{
					Kind:    adapters.EventAudio,
					Speaker: adapters.SpeakerCharacter,
					Audio:   raw,
					StartMs: ev.StartMs,
					EndMs:   ev.EndMs,
				})
			}

		case "session.input_audio.append":
			// Reflected player audio. The browser sent it; this is our copy.
			if raw, err := base64.StdEncoding.DecodeString(ev.Audio); err == nil {
				c.emit(adapters.LiveEvent{
					Kind:    adapters.EventAudio,
					Speaker: adapters.SpeakerPlayer,
					Audio:   raw,
				})
			}

		case "session.usage.updated":
			// A running total, not an increment: the documentation is explicit
			// that these must not be summed.
			c.emit(adapters.LiveEvent{
				Kind: adapters.EventUsage,
				// The resolved model name, never a tier: a price attaches to
				// a model.
				Usage: adapters.Usage{Model: c.modelID, AudioSeconds: ev.Usage.Seconds},
			})

		case "session.instructions.appended":
			c.settle(ev.ClientEventID)

		case "session.closed":
			c.settleAll()
			// Released before the event is emitted, so a Close waiting on it is
			// not held behind a reader that has already stopped listening.
			c.finish()
			c.emit(adapters.LiveEvent{Kind: adapters.EventClosed, CloseReason: ev.Reason})
			return

		case "error":
			c.settle(ev.ClientEventID)
			c.emit(adapters.LiveEvent{
				Kind: adapters.EventError,
				Err:  fmt.Errorf("gpt-live: %s: %s", ev.Error.Code, ev.Error.Message),
			})
		}
	}
}

func (c *liveConn) settle(id string) {
	if id == "" {
		return
	}
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// finish marks the session finalised, at most once.
func (c *liveConn) finish() {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.finalised:
	default:
		close(c.finalised)
	}
}

func (c *liveConn) settleAll() {
	c.mu.Lock()
	clear(c.pending)
	c.mu.Unlock()
}

func (c *liveConn) emit(e adapters.LiveEvent) {
	select {
	case c.events <- e:
	case <-c.done:
	}
}

func readError(resp *http.Response) (string, error) {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Error.Message == "" {
		return "", nil
	}
	return ": " + body.Error.Message, nil
}
