package mistral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"engine/internal/adapters"
)

// Thread continues a conversation through the Conversations API: the first
// call starts a stored conversation, each later one appends to it, and the
// reply streams as server-sent events.
//
// Appending takes neither a model nor instructions, and an input entry may only
// be a user's or an assistant's. A conversation therefore stays on the model it
// started with, and holds the System Message as the instructions it started
// with. Having no developer role, a call's constraints travel in its user
// entry, after the input.
type Thread struct {
	keys  adapters.KeyFunc
	model string
	api   client
}

func NewThread(keys adapters.KeyFunc, model string) *Thread {
	return &Thread{
		keys:  keys,
		model: model,
		// A story reply streams for a while; this bounds a call that hangs.
		api: client{base: baseURL, http: &http.Client{Timeout: 3 * time.Minute}},
	}
}

func (t *Thread) Model() string { return t.model }

// entry is one message in a conversation's inputs.
type entry struct {
	Object  string `json:"object"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

func message(role, content string) entry {
	return entry{Object: "entry", Type: "message.input", Role: role, Content: content}
}

func (t *Thread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	key, err := t.keys(ctx)
	if err != nil {
		return adapters.ThreadReply{}, fmt.Errorf("resolve api key: %w", err)
	}

	now := userEntry(turn.Exchange("", ""))
	// Sent on start and on append, as a format is the call's own: a plain turn
	// on a conversation that answered structured before still answers in text.
	args := map[string]any{"response_format": responseFormat(turn.Schema)}

	if id := turn.Continuation.ID; id != "" {
		reply, err := t.call(ctx, key, "/conversations/"+id, map[string]any{
			"inputs":          []entry{message("user", now)},
			"completion_args": args,
			"store":           true,
			"stream":          true,
		}, turn)
		// Mistral no longer holding the conversation is answered by a fresh
		// one, seeded with the transcript; anything else is the turn's failure.
		if !errors.Is(err, errNotFound) {
			return reply, err
		}
	}

	inputs := make([]entry, 0, 2*len(turn.Transcript)+2)
	if turn.Summary != "" {
		// The game's own message, which Mistral takes as the user's.
		inputs = append(inputs, message("user", turn.Summary))
	}
	for _, e := range turn.Transcript {
		inputs = append(inputs, message("user", userEntry(e)), message("assistant", e.Reply))
	}
	start := map[string]any{
		"model":           t.model,
		"inputs":          append(inputs, message("user", now)),
		"completion_args": args,
		"store":           true,
		"stream":          true,
	}
	if turn.System != "" {
		start["instructions"] = turn.System
	}
	return t.call(ctx, key, "/conversations", start, turn)
}

// responseFormat is the Conversations API's format for a turn.
func responseFormat(schema *adapters.Schema) map[string]any {
	if schema == nil {
		return map[string]any{"type": "text"}
	}
	return map[string]any{
		"type":        "json_schema",
		"json_schema": map[string]any{"name": schema.Name, "schema": schema.Body, "strict": true},
	}
}

// userEntry is the user entry an exchange sends: its input, with the constraints
// after it, as there is no message of their own for them.
func userEntry(e adapters.Exchange) string {
	if e.Constraints == "" {
		return e.Input
	}
	return e.Input + "\n\n" + e.Constraints
}

// streamEvent is the part of each server-sent event the thread reads.
type streamEvent struct {
	Type           string          `json:"type"`
	ConversationID string          `json:"conversation_id"`
	Content        json.RawMessage `json:"content"`
	Message        string          `json:"message"`
	Code           json.RawMessage `json:"code"`
	Usage          *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

// call makes one streamed request and follows it to the finished response.
func (t *Thread) call(ctx context.Context, key, path string, request map[string]any, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return adapters.ThreadReply{}, err
	}
	resp, err := t.api.send(ctx, key, http.MethodPost, path, body, "text/event-stream")
	if err != nil {
		return adapters.ThreadReply{}, err
	}
	defer resp.Body.Close()

	var (
		text strings.Builder
		id   string
		done *streamEvent
	)
	err = adapters.ReadEvents("mistral conversations stream", resp.Body, func(data []byte) error {
		var e streamEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("mistral conversations: unreadable event: %w", err)
		}
		switch e.Type {
		case "conversation.response.started":
			id = e.ConversationID
		case "message.output.delta":
			piece := pieceOf(e.Content)
			text.WriteString(piece)
			turn.Hear(piece)
		case "conversation.response.done":
			done = &e
		case "conversation.response.error":
			// Mistral names no codes for these, so a failure inside the
			// stream is treated as one that may pass.
			return fmt.Errorf("mistral conversations: %s (%s): %w", e.Message, e.Code, adapters.ErrTransient)
		}
		return nil
	})
	if err != nil {
		return adapters.ThreadReply{}, err
	}
	if done == nil {
		return adapters.ThreadReply{}, adapters.Transient("mistral conversations stream", io.ErrUnexpectedEOF)
	}
	// A plain turn's empty reply fails; a structured turn's is its caller's to
	// judge, as it judges any reply against the schema.
	if text.Len() == 0 && turn.Schema == nil {
		return adapters.ThreadReply{}, fmt.Errorf("mistral conversations: empty output")
	}
	if id == "" {
		// An append answers on the conversation it was sent to.
		appended, ok := strings.CutPrefix(path, "/conversations/")
		if !ok {
			return adapters.ThreadReply{}, fmt.Errorf("mistral conversations: no conversation id")
		}
		id = appended
	}

	used := adapters.Usage{Model: t.model}
	if done.Usage != nil {
		used.InputTokens = done.Usage.PromptTokens
		used.OutputTokens = done.Usage.CompletionTokens
	}
	return adapters.ThreadReply{
		Text:         text.String(),
		Usage:        used,
		Continuation: adapters.Continuation{ID: id, Model: t.model, Made: time.Now().UTC()},
	}, nil
}

// pieceOf reads a delta's content, a string for text or a chunk once a tool
// has produced something.
func pieceOf(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var c chunk
	if json.Unmarshal(raw, &c) == nil && c.Type == "text" {
		return c.Text
	}
	return ""
}
