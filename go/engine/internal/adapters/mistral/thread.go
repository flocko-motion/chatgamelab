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
// started with, and since the blocks sharing a thread each have their own
// instructions, a call's instructions travel in its user entry, ahead of the
// input.
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

	said := withInstructions(turn.Instructions, turn.Input)

	if id := turn.Continuation.ID; id != "" {
		reply, err := t.call(ctx, key, "/conversations/"+id, map[string]any{
			"inputs": []entry{message("user", said)},
			"store":  true,
			"stream": true,
		}, turn)
		// Mistral no longer holding the conversation is answered by a fresh
		// one, seeded with the transcript; anything else is the turn's failure.
		if !errors.Is(err, errNotFound) {
			return reply, err
		}
	}

	inputs := make([]entry, 0, 2*len(turn.Transcript)+1)
	for _, e := range turn.Transcript {
		inputs = append(inputs, message("user", withInstructions(e.Instructions, e.Input)), message("assistant", e.Reply))
	}
	return t.call(ctx, key, "/conversations", map[string]any{
		"model":  t.model,
		"inputs": append(inputs, message("user", said)),
		"store":  true,
		"stream": true,
	}, turn)
}

// withInstructions puts a call's instructions ahead of its input, in the one
// user entry that can carry them.
func withInstructions(instructions, input string) string {
	if instructions == "" {
		return input
	}
	return instructions + "\n\n" + input
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
	if text.Len() == 0 {
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
