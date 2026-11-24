package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"engine/internal/adapters"
)

// realtimeURL is the Realtime API's server-side WebSocket; the model goes in
// the query.
const realtimeURL = "wss://api.openai.com/v1/realtime"

// sessionLimit is how long OpenAI keeps a Realtime session open. A session
// within sessionMargin of it is replaced before a call, so no reply is cut
// off by the limit.
const (
	sessionLimit  = 60 * time.Minute
	sessionMargin = 5 * time.Minute
)

// RealtimeVoices are the voices a gpt-realtime model speaks in, as the
// provider lists them, with the presentation listeners hear; alloy's is not
// stated. Each speaks English natively and follows the conversation into
// other languages.
var RealtimeVoices = []adapters.BuiltinVoice{
	{Name: "marin", Language: "en", Accent: "North American", Gender: "feminine"},
	{Name: "cedar", Language: "en", Accent: "North American", Gender: "masculine"},
	{Name: "alloy", Language: "en", Accent: "North American"},
	{Name: "ash", Language: "en", Accent: "North American", Gender: "masculine"},
	{Name: "ballad", Language: "en", Accent: "British", Gender: "masculine"},
	{Name: "coral", Language: "en", Accent: "North American", Gender: "feminine"},
	{Name: "echo", Language: "en", Accent: "North American", Gender: "masculine"},
	{Name: "sage", Language: "en", Accent: "North American", Gender: "feminine"},
	{Name: "shimmer", Language: "en", Accent: "North American", Gender: "feminine"},
	{Name: "verse", Language: "en", Accent: "North American", Gender: "masculine"},
}

// Realtime continues a conversation on a gpt-realtime model, a native
// audio-out model: the model that writes a reply speaks it. It talks to the
// Realtime API over the server-side WebSocket, with the standard API key.
//
// The continuation is the session itself, which the adapter holds open
// between calls; its handle names the session and the turns kept in it. A
// handle the adapter does not hold, as after an engine restart, a dropped
// connection or a session near its limit, opens a fresh session seeded with
// the transcript. A reply the caller did not keep is deleted from the session
// when the next call goes on from the handle before it.
//
// The System Message is the session's instructions, set as it opens, and set
// again on a session seeded fresh. A call's constraints go in a system message
// of their own, right before its input, which is the game's own system message
// or the player's user message.
//
// A spoken turn asks for audio, which comes with its transcript; a plain turn
// asks for text. A JSON turn is a forced function call, named after the
// schema with the schema as its parameters, its arguments the reply: Realtime
// offers no JSON schema format and no strict mode, so the caller checks the
// reply against the schema, as it does any reply.
type Realtime struct {
	keys  adapters.KeyFunc
	model string
	voice string
	// url and now are fields so a test can point them at a fake.
	url string
	now func() time.Time

	// mu holds one call at a time on the session.
	mu      sync.Mutex
	session *realtimeSession
}

// NewRealtime builds the adapter; voice is the one a turn without its own
// speaks in, an empty one DefaultVoice.
func NewRealtime(keys adapters.KeyFunc, model, voice string) *Realtime {
	if voice == "" {
		voice = DefaultVoice
	}
	return &Realtime{keys: keys, model: model, voice: voice, url: realtimeURL, now: time.Now}
}

func (r *Realtime) Model() string { return r.model }

// Close ends the session the adapter holds, if any.
func (r *Realtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drop()
	return nil
}

func (r *Realtime) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	if turn.Speak && turn.Schema != nil {
		return adapters.ThreadReply{}, errors.New("realtime api: a spoken turn takes no schema")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	s, err := r.ready(ctx, turn)
	if err != nil {
		return adapters.ThreadReply{}, err
	}
	reply, err := s.respond(ctx, turn)
	if reply.Usage != (adapters.Usage{}) {
		// The model, never a tier: a price attaches to a model.
		reply.Usage.Model = r.model
	}
	switch {
	case errors.Is(err, adapters.ErrBadReply):
		// The provider answered, and its reply goes; the session stays as
		// it was before the call.
		if s.forget(ctx, s.pending) != nil {
			r.drop()
		}
		return reply, err
	case err != nil:
		// What the provider holds after a failure is unknown, so the session
		// goes and the next call seeds a fresh one.
		r.drop()
		return reply, err
	}
	s.keep()
	reply.Continuation = adapters.Continuation{ID: s.handle(s.turns), Model: r.model, Made: r.now().UTC()}
	return reply, nil
}

// ready is the session a turn goes on in: the one held, if the turn's
// continuation names it, or a fresh one seeded with the transcript.
func (r *Realtime) ready(ctx context.Context, turn adapters.ThreadTurn) (*realtimeSession, error) {
	// The provider fixes a session's voice: another voice takes a fresh one.
	sameVoice := func(s *realtimeSession) bool { return turn.Voice == "" || turn.Voice == s.voice }
	if s := r.session; s != nil && sameVoice(s) && r.now().Sub(s.opened) < sessionLimit-sessionMargin {
		switch turn.Continuation.ID {
		case s.handle(s.turns):
			return s, nil
		case s.handle(s.turns - 1):
			// The last reply was not kept, so it leaves the session.
			if s.last != nil && s.forget(ctx, s.last) == nil {
				s.turns--
				s.last = nil
				return s, nil
			}
		}
	}
	r.drop()
	voice := turn.Voice
	if voice == "" {
		voice = r.voice
	}
	s, err := r.open(ctx, voice, turn.System, turn.Summary, turn.Transcript)
	if err != nil {
		return nil, err
	}
	r.session = s
	return s, nil
}

// drop closes the session held, if any.
func (r *Realtime) drop() {
	if r.session != nil {
		r.session.close()
		r.session = nil
	}
}

// open connects a fresh session, configures it with the System Message, and
// seeds it with the summary, as the game's system message, and the
// transcript, each exchange as it was sent.
func (r *Realtime) open(ctx context.Context, voice, system, summary string, transcript []adapters.Exchange) (*realtimeSession, error) {
	key, err := r.keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve api key: %w", err)
	}
	ws, resp, err := websocket.Dial(ctx, r.url+"?model="+url.QueryEscape(r.model), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + key}},
	})
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			return nil, adapters.StatusError("realtime api", resp, "")
		}
		return nil, adapters.Transient("realtime api", err)
	}
	// No read limit: an audio delta is large.
	ws.SetReadLimit(-1)

	s := &realtimeSession{ws: ws, voice: voice, opened: r.now(), events: make(chan realtimeEvent, 256), gone: make(chan struct{})}
	go s.read()

	created, err := s.next(ctx)
	if err == nil && created.Type != "session.created" {
		err = fmt.Errorf("realtime api: the session opened with %s", created.Type)
	}
	if err != nil {
		s.close()
		return nil, err
	}
	s.id = created.Session.ID

	// Text in, no voice activity detection: the engine sends whole turns
	// and asks for each response itself.
	err = s.send(ctx, map[string]any{"type": "session.update", "session": map[string]any{
		"type":              "realtime",
		"instructions":      system,
		"output_modalities": []string{"audio"},
		"audio": map[string]any{
			"input":  map[string]any{"turn_detection": nil},
			"output": map[string]any{"voice": voice, "format": map[string]any{"type": "audio/pcm", "rate": 24000}},
		},
	}})
	if err == nil && summary != "" {
		err = s.add(ctx, textItem("system", "input_text", summary))
	}
	for _, e := range transcript {
		if err != nil {
			break
		}
		if _, err = s.say(ctx, e); err == nil {
			err = s.add(ctx, textItem("assistant", "output_text", e.Reply))
		}
	}
	if err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

// realtimeSession is one open session and what it holds.
type realtimeSession struct {
	ws *websocket.Conn
	id string
	// voice is the voice it speaks in, fixed as it opens.
	voice  string
	opened time.Time

	// turns is how many turns the session has kept, the handle's number.
	turns int
	// last are the items the last kept turn added, which forgetting it
	// deletes; pending are those of the turn running.
	last, pending []string
	// items numbers the items the engine creates.
	items int

	// events is the server's events, read as they come so the connection
	// answers its pings between calls; gone closes when reading stops, err
	// saying why.
	events chan realtimeEvent
	gone   chan struct{}
	err    error
}

// handle is the continuation ID of the session after n turns.
func (s *realtimeSession) handle(n int) string { return s.id + "/" + strconv.Itoa(n) }

// keep counts the turn just made as kept.
func (s *realtimeSession) keep() {
	s.turns++
	s.last, s.pending = s.pending, nil
}

func (s *realtimeSession) close() { _ = s.ws.Close(websocket.StatusNormalClosure, "") }

// realtimeEvent is the part of a server event the adapter reads.
type realtimeEvent struct {
	Type    string `json:"type"`
	Delta   string `json:"delta"`
	ItemID  string `json:"item_id"`
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Response realtimeResponse `json:"response"`
}

type realtimeResponse struct {
	Status        string `json:"status"`
	StatusDetails struct {
		Reason string `json:"reason"`
		Error  struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"status_details"`
	Output []struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		CallID    string `json:"call_id"`
		Arguments string `json:"arguments"`
		Content   []struct {
			Type       string `json:"type"`
			Text       string `json:"text"`
			Transcript string `json:"transcript"`
		} `json:"content"`
	} `json:"output"`
	Usage struct {
		InputTokens       int64 `json:"input_tokens"`
		OutputTokens      int64 `json:"output_tokens"`
		InputTokenDetails struct {
			TextTokens          int64 `json:"text_tokens"`
			AudioTokens         int64 `json:"audio_tokens"`
			CachedTokensDetails struct {
				TextTokens  int64 `json:"text_tokens"`
				AudioTokens int64 `json:"audio_tokens"`
			} `json:"cached_tokens_details"`
		} `json:"input_token_details"`
		OutputTokenDetails struct {
			TextTokens  int64 `json:"text_tokens"`
			AudioTokens int64 `json:"audio_tokens"`
		} `json:"output_token_details"`
	} `json:"usage"`
}

// read hands on the server's events until the connection ends.
func (s *realtimeSession) read() {
	defer close(s.gone)
	for {
		_, data, err := s.ws.Read(context.Background())
		if err != nil {
			s.err = err
			return
		}
		var e realtimeEvent
		if json.Unmarshal(data, &e) != nil {
			continue
		}
		s.events <- e
	}
}

// next is the server's next event.
func (s *realtimeSession) next(ctx context.Context) (realtimeEvent, error) {
	select {
	case e := <-s.events:
		return e, nil
	case <-s.gone:
		// Events read before the end still come first.
		select {
		case e := <-s.events:
			return e, nil
		default:
		}
		err := s.err
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return realtimeEvent{}, adapters.Transient("realtime api", err)
	case <-ctx.Done():
		return realtimeEvent{}, ctx.Err()
	}
}

func (s *realtimeSession) send(ctx context.Context, event map[string]any) error {
	blob, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := s.ws.Write(ctx, websocket.MessageText, blob); err != nil {
		return adapters.Transient("realtime api", err)
	}
	return nil
}

// add creates an item in the conversation.
func (s *realtimeSession) add(ctx context.Context, item map[string]any) error {
	_, err := s.addItem(ctx, item)
	return err
}

// addItem creates an item under an ID of the engine's, so it can be deleted
// later, and returns the ID.
func (s *realtimeSession) addItem(ctx context.Context, item map[string]any) (string, error) {
	s.items++
	id := "cgl_" + strconv.Itoa(s.items)
	item["id"] = id
	return id, s.send(ctx, map[string]any{"type": "conversation.item.create", "item": item})
}

// say adds what an exchange sends: its constraints in a system message of
// their own, then its input, as the game's system message or the player's
// user message. It returns the items it added.
func (s *realtimeSession) say(ctx context.Context, e adapters.Exchange) ([]string, error) {
	var added []string
	if e.Constraints != "" {
		id, err := s.addItem(ctx, textItem("system", "input_text", e.Constraints))
		if err != nil {
			return added, err
		}
		added = append(added, id)
	}
	role := "user"
	if e.Directive {
		role = "system"
	}
	id, err := s.addItem(ctx, textItem(role, "input_text", e.Input))
	if err != nil {
		return added, err
	}
	return append(added, id), nil
}

// textItem is a conversation item of one text part.
func textItem(role, part, text string) map[string]any {
	return map[string]any{
		"type":    "message",
		"role":    role,
		"content": []map[string]any{{"type": part, "text": text}},
	}
}

// forget deletes items from the conversation, and waits until they are gone.
func (s *realtimeSession) forget(ctx context.Context, items []string) error {
	waiting := map[string]bool{}
	for _, id := range items {
		if err := s.send(ctx, map[string]any{"type": "conversation.item.delete", "item_id": id}); err != nil {
			return err
		}
		waiting[id] = true
	}
	for len(waiting) > 0 {
		e, err := s.next(ctx)
		if err != nil {
			return err
		}
		switch e.Type {
		case "conversation.item.deleted":
			delete(waiting, e.ItemID)
		case "error":
			return eventError(e.Error.Type, e.Error.Code, e.Error.Message)
		}
	}
	return nil
}

// respond makes one turn in the session: what the turn sends, then a
// response, read to its end. The items it adds are pending until it is kept.
// The response sets no instructions, which would replace the session's.
func (s *realtimeSession) respond(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	added, err := s.say(ctx, turn.Exchange("", ""))
	s.pending = added
	if err != nil {
		return adapters.ThreadReply{}, err
	}

	response := map[string]any{"output_modalities": []string{"text"}}
	switch {
	case turn.Schema != nil:
		response["tools"] = []map[string]any{{"type": "function", "name": turn.Schema.Name, "parameters": turn.Schema.Body}}
		response["tool_choice"] = map[string]any{"type": "function", "name": turn.Schema.Name}
	case turn.Speak:
		response["output_modalities"] = []string{"audio"}
	}
	if err := s.send(ctx, map[string]any{"type": "response.create", "response": response}); err != nil {
		return adapters.ThreadReply{}, err
	}

	var text strings.Builder
	for {
		e, err := s.next(ctx)
		if err != nil {
			return adapters.ThreadReply{}, err
		}
		switch e.Type {
		case "response.output_audio_transcript.delta", "response.output_text.delta", "response.function_call_arguments.delta":
			text.WriteString(e.Delta)
			turn.Hear(e.Delta)
		case "response.output_audio.delta":
			if chunk, err := base64.StdEncoding.DecodeString(e.Delta); err == nil {
				turn.HearAudio(chunk)
			}
		case "error":
			return adapters.ThreadReply{}, eventError(e.Error.Type, e.Error.Code, e.Error.Message)
		case "response.done":
			return s.finish(ctx, turn, e.Response, text.String())
		}
	}
}

// finish reads a done response into the reply.
func (s *realtimeSession) finish(ctx context.Context, turn adapters.ThreadTurn, done realtimeResponse, streamed string) (adapters.ThreadReply, error) {
	used := realtimeUsed(done)
	reply, call := streamed, ""
	for _, item := range done.Output {
		s.pending = append(s.pending, item.ID)
		switch item.Type {
		case "function_call":
			reply, call = item.Arguments, item.CallID
		case "message":
			if streamed != "" {
				continue
			}
			for _, part := range item.Content {
				reply += part.Text + part.Transcript
			}
		}
	}

	switch done.Status {
	case "completed":
	case "incomplete":
		why := done.StatusDetails.Reason
		if why == "content_filter" {
			return adapters.ThreadReply{Usage: used}, fmt.Errorf("realtime api: %s: %w", why, adapters.ErrRefused)
		}
		return adapters.ThreadReply{Usage: used}, fmt.Errorf("realtime api: %s: %w", why, adapters.ErrTruncated)
	case "cancelled":
		return adapters.ThreadReply{Usage: used}, adapters.Transient("realtime api", errors.New("the response was cancelled"))
	default:
		failure := done.StatusDetails.Error
		return adapters.ThreadReply{Usage: used}, eventError(failure.Type, failure.Code, failure.Message)
	}
	if reply == "" && turn.Schema == nil {
		// A plain turn's empty reply fails; a structured turn's is its
		// caller's to judge, as it judges any reply against the schema.
		return adapters.ThreadReply{Usage: used}, errors.New("realtime api: empty output")
	}
	if call != "" {
		// A function call answered keeps the conversation whole for the
		// turns after it.
		output, err := s.addItem(ctx, map[string]any{"type": "function_call_output", "call_id": call, "output": `{"ok":true}`})
		if err != nil {
			return adapters.ThreadReply{Usage: used}, err
		}
		s.pending = append(s.pending, output)
	}
	return adapters.ThreadReply{Text: reply, Usage: used}, nil
}

// realtimeUsed reads a done response's usage, cached input and audio apart,
// as each is priced apart.
func realtimeUsed(done realtimeResponse) adapters.Usage {
	in, out := done.Usage.InputTokenDetails, done.Usage.OutputTokenDetails
	cached := in.CachedTokensDetails
	return adapters.Usage{
		InputTokens:            in.TextTokens - cached.TextTokens,
		CachedInputTokens:      cached.TextTokens,
		OutputTokens:           out.TextTokens,
		AudioInputTokens:       in.AudioTokens - cached.AudioTokens,
		CachedAudioInputTokens: cached.AudioTokens,
		AudioOutputTokens:      out.AudioTokens,
	}
}

// eventError is an error the provider reported; a server error or a rate
// limit may pass.
func eventError(kind, code, message string) error {
	err := fmt.Errorf("realtime api: %s: %s", strings.TrimSpace(kind+" "+code), message)
	if strings.Contains(kind, "server_error") || strings.Contains(code, "server_error") || strings.Contains(code, "rate_limit") {
		return fmt.Errorf("%w: %w", err, adapters.ErrTransient)
	}
	return err
}
