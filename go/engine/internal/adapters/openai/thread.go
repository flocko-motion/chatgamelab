package openai

import (
	"bytes"
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

// retention is how long OpenAI keeps a stored response; a continuation older
// than this names a response that is gone.
const retention = 30 * 24 * time.Hour

// Thread continues a conversation through the Responses API: each call names
// the previous response, and OpenAI holds what came before. Responses are
// stored, which a continuation needs, and the reply streams as server-sent
// events.
type Thread struct {
	keys  adapters.KeyFunc
	model string
	// url is a field so a test can point it at a fake server.
	url    string
	client *http.Client
	now    func() time.Time
}

func NewThread(keys adapters.KeyFunc, model string) *Thread {
	return &Thread{
		keys:  keys,
		model: model,
		url:   responsesURL,
		// A story reply streams for a while; this bounds a call that hangs.
		client: &http.Client{Timeout: 3 * time.Minute},
		now:    time.Now,
	}
}

func (t *Thread) Model() string { return t.model }

// errGone is the provider answering that the previous response does not exist.
var errGone = errors.New("the previous response is gone")

func (t *Thread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	key, err := t.keys(ctx)
	if err != nil {
		return adapters.ThreadReply{}, fmt.Errorf("resolve api key: %w", err)
	}

	from := turn.Continuation.ID
	if from != "" && t.now().Sub(turn.Continuation.Made) >= retention {
		from = ""
	}
	reply, err := t.call(ctx, key, turn, from)
	if errors.Is(err, errGone) {
		// Nothing was said on the lost thread, so a fresh one seeded with the
		// transcript takes the same turn.
		reply, err = t.call(ctx, key, turn, "")
	}
	return reply, err
}

type turnMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// sent is what one exchange sends: the constraints in a developer message of
// their own, right before the input, so designer text never shares their
// message; then the input, as the game's developer message or the player's.
func sent(e adapters.Exchange) []turnMessage {
	var out []turnMessage
	if e.Constraints != "" {
		out = append(out, turnMessage{Role: "developer", Content: e.Constraints})
	}
	role := "user"
	if e.Directive {
		role = "developer"
	}
	return append(out, turnMessage{Role: role, Content: e.Input})
}

// call makes one streamed request, continuing from the response named, or on a
// fresh thread seeded with the transcript when none is.
//
// The System Message opens a fresh thread as its first developer message,
// which is stored with the thread; OpenAI drops instructions over a previous
// response. A summary follows it, as the game's developer message.
func (t *Thread) call(ctx context.Context, key string, turn adapters.ThreadTurn, from string) (adapters.ThreadReply, error) {
	request := map[string]any{
		"model":  t.model,
		"store":  true,
		"stream": true,
		// Sent on every call, as a format is the call's own: a plain turn on a
		// thread that answered structured before still answers in text.
		"text": map[string]any{"format": format(turn.Schema)},
	}
	now := sent(turn.Exchange("", ""))
	if from != "" {
		request["previous_response_id"] = from
		request["input"] = now
	} else {
		var input []turnMessage
		if turn.System != "" {
			input = append(input, turnMessage{Role: "developer", Content: turn.System})
		}
		if turn.Summary != "" {
			input = append(input, turnMessage{Role: "developer", Content: turn.Summary})
		}
		for _, e := range turn.Transcript {
			input = append(append(input, sent(e)...), turnMessage{Role: "assistant", Content: e.Reply})
		}
		request["input"] = append(input, now...)
	}
	body, err := json.Marshal(request)
	if err != nil {
		return adapters.ThreadReply{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return adapters.ThreadReply{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.client.Do(req)
	if err != nil {
		return adapters.ThreadReply{}, adapters.Transient("responses api", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if from != "" && gone(resp.StatusCode, detail) {
			return adapters.ThreadReply{}, errGone
		}
		return adapters.ThreadReply{}, adapters.StatusError("responses api", resp, ": "+strings.TrimSpace(string(detail)))
	}
	return t.read(resp.Body, turn)
}

// format is the Responses API's text format for a turn.
func format(schema *adapters.Schema) map[string]any {
	if schema == nil {
		return map[string]any{"type": "text"}
	}
	return map[string]any{"type": "json_schema", "name": schema.Name, "schema": schema.Body, "strict": true}
}

// gone reports whether a refusal says the previous response does not exist.
func gone(status int, detail []byte) bool {
	return status == http.StatusNotFound || bytes.Contains(detail, []byte("previous_response_not_found"))
}

// streamEvent is the part of each server-sent event the thread reads.
type streamEvent struct {
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Response struct {
		ID                string `json:"id"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Usage struct {
			InputTokens        int64 `json:"input_tokens"`
			OutputTokens       int64 `json:"output_tokens"`
			InputTokensDetails struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
	} `json:"response"`
}

// read follows the stream to its completed response, handing on each piece.
func (t *Thread) read(body io.Reader, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	var (
		text    strings.Builder
		done    *streamEvent
		refused bool
	)
	err := adapters.ReadEvents("responses api stream", body, func(data []byte) error {
		var e streamEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("responses api: unreadable event: %w", err)
		}
		switch e.Type {
		case "response.output_text.delta":
			text.WriteString(e.Delta)
			turn.Hear(e.Delta)
		case "response.refusal.delta":
			refused = true
		case "response.completed", "response.incomplete":
			// An incomplete response is finished but cut short: the provider
			// is fine, the reply cannot be used.
			done = &e
		case "response.failed":
			code, msg := e.Type, ""
			if e.Response.Error != nil {
				code, msg = e.Response.Error.Code, e.Response.Error.Message
			}
			return streamFailure(code, msg)
		case "error":
			return streamFailure(e.Code, e.Message)
		}
		return nil
	})
	if err != nil {
		return adapters.ThreadReply{}, err
	}
	if done == nil {
		return adapters.ThreadReply{}, adapters.Transient("responses api stream", io.ErrUnexpectedEOF)
	}

	usage := done.Response.Usage
	cached := usage.InputTokensDetails.CachedTokens
	used := adapters.Usage{
		Model:             t.model,
		InputTokens:       usage.InputTokens - cached,
		CachedInputTokens: cached,
		OutputTokens:      usage.OutputTokens,
	}
	// A bad reply is billed all the same, so its usage comes back with it.
	switch {
	case done.Type == "response.incomplete":
		why := "incomplete"
		if d := done.Response.IncompleteDetails; d != nil && d.Reason != "" {
			why = d.Reason
		}
		return adapters.ThreadReply{Usage: used}, fmt.Errorf("responses api: %s: %w", why, adapters.ErrTruncated)
	case refused:
		return adapters.ThreadReply{Usage: used}, fmt.Errorf("responses api: %w", adapters.ErrRefused)
	case text.Len() == 0 && turn.Schema == nil:
		// A plain turn's empty reply fails; a structured turn's is its
		// caller's to judge, as it judges any reply against the schema.
		return adapters.ThreadReply{}, fmt.Errorf("responses api: empty output")
	}
	return adapters.ThreadReply{
		Text:         text.String(),
		Usage:        used,
		Continuation: adapters.Continuation{ID: done.Response.ID, Model: t.model, Made: t.now().UTC()},
	}, nil
}

// streamFailure is a response the provider reported failed partway; one that
// may pass, such as an overloaded server, is made again.
func streamFailure(code, message string) error {
	err := fmt.Errorf("responses api: %s: %s", code, message)
	if strings.Contains(code, "rate_limit") || strings.Contains(code, "server_error") {
		return fmt.Errorf("%w: %w", err, adapters.ErrTransient)
	}
	return err
}
