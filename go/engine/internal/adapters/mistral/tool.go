package mistral

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"engine/internal/adapters"
)

// Tool is one single-shot text call; the role is the prompt, not the type.
type Tool struct {
	keys  adapters.KeyFunc
	model string
	api   client
}

func NewTool(keys adapters.KeyFunc, model string) *Tool {
	return &Tool{
		keys:  keys,
		model: model,
		// The observer runs beside speech, so it must stay well under one
		// spoken utterance to go unnoticed.
		api: client{base: BaseURL, http: &http.Client{Timeout: 10 * time.Second}},
	}
}

func (t *Tool) Query(ctx context.Context, system, user string) (string, adapters.Usage, error) {
	return t.query(ctx, system, user, nil)
}

// QuerySchema asks for JSON of the schema, which the Conversations API holds
// the model to in strict mode.
func (t *Tool) QuerySchema(ctx context.Context, system, user string, schema adapters.Schema) (string, adapters.Usage, error) {
	return t.query(ctx, system, user, &schema)
}

// QueryImage shows the model a picture after the user's words, and asks for
// JSON of the schema.
func (t *Tool) QueryImage(ctx context.Context, system, user string, image []byte, schema adapters.Schema) (string, adapters.Usage, error) {
	content := []map[string]any{
		{"type": "text", "text": user},
		{"type": "image_url", "image_url": adapters.DataURL(image)},
	}
	return t.query(ctx, system, content, &schema)
}

// query sends one input: plain words, or a list of chunks.
func (t *Tool) query(ctx context.Context, system string, user any, schema *adapters.Schema) (string, adapters.Usage, error) {
	used := adapters.Usage{Model: t.model}

	key, err := t.keys(ctx)
	if err != nil {
		return "", used, fmt.Errorf("resolve api key: %w", err)
	}

	out, err := t.api.converse(ctx, key, start{
		Model:        t.model,
		Instructions: system,
		Inputs:       []input{{Role: "user", Content: user}},
		CompletionArgs: func() map[string]any {
			if schema == nil {
				return nil
			}
			return map[string]any{"response_format": responseFormat(schema)}
		}(),
	})
	if err != nil {
		return "", used, err
	}
	used.InputTokens = out.Usage.PromptTokens
	used.OutputTokens = out.Usage.CompletionTokens

	text := out.text()
	if text == "" {
		return "", used, fmt.Errorf("mistral conversations: empty output")
	}
	return text, used, nil
}
