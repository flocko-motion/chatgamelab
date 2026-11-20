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
		api: client{base: baseURL, http: &http.Client{Timeout: 10 * time.Second}},
	}
}

func (t *Tool) Query(ctx context.Context, system, user string) (string, adapters.Usage, error) {
	used := adapters.Usage{Model: t.model}

	key, err := t.keys(ctx)
	if err != nil {
		return "", used, fmt.Errorf("resolve api key: %w", err)
	}

	out, err := t.api.converse(ctx, key, start{
		Model:        t.model,
		Instructions: system,
		Inputs:       []input{{Role: "user", Content: user}},
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
