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
	"time"

	"engine/internal/adapters"
)

const chatURL = "https://api.openai.com/v1/chat/completions"

// Hearing hears a clip of the player's speech on an audio model, such as
// gpt-audio-mini, through Chat Completions: one plain HTTP call per clip, the
// clip sent as WAV. The model answers in text only.
//
// The reply is a forced function call, named after the schema with the
// schema as its parameters, its arguments the reply: the audio models offer
// no JSON schema format, so the caller checks the arguments against the
// schema, as it does any reply.
type Hearing struct {
	keys  adapters.KeyFunc
	model string
	// url is a field so a test can point it at a fake.
	url    string
	client *http.Client
}

func NewHearing(keys adapters.KeyFunc, model string) *Hearing {
	return &Hearing{keys: keys, model: model, url: chatURL, client: &http.Client{Timeout: 30 * time.Second}}
}

func (h *Hearing) Hear(ctx context.Context, system string, clip adapters.Clip, schema adapters.Schema) (string, adapters.Usage, error) {
	used := adapters.Usage{Model: h.model}
	key, err := h.keys(ctx)
	if err != nil {
		return "", used, fmt.Errorf("resolve api key: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"model":      h.model,
		"modalities": []string{"text"},
		"messages": []map[string]any{
			{"role": "system", "content": system},
			{"role": "user", "content": []map[string]any{{
				"type":        "input_audio",
				"input_audio": map[string]string{"data": base64.StdEncoding.EncodeToString(clip.WAV()), "format": "wav"},
			}}},
		},
		"tools": []map[string]any{{
			"type":     "function",
			"function": map[string]any{"name": schema.Name, "parameters": schema.Body},
		}},
		"tool_choice": map[string]any{"type": "function", "function": map[string]string{"name": schema.Name}},
	})
	if err != nil {
		return "", used, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return "", used, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", used, adapters.Transient("chat completions", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", used, adapters.StatusError("chat completions", resp, ": "+strings.TrimSpace(string(detail)))
	}

	var out chatReply
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", used, adapters.Transient("chat completions", err)
	}
	args, err := out.arguments(schema.Name)
	return args, out.usage(h.model), err
}

// chatReply is the part of a Chat Completions reply the adapter reads.
type chatReply struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Refusal   string `json:"refusal"`
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int64 `json:"prompt_tokens"`
		CompletionTokens    int64 `json:"completion_tokens"`
		PromptTokensDetails struct {
			CachedTokens int64 `json:"cached_tokens"`
			AudioTokens  int64 `json:"audio_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
}

// usage counts the clip's audio tokens apart from the text, and cached input
// apart from the rest, as each is priced apart.
func (r chatReply) usage(model string) adapters.Usage {
	details := r.Usage.PromptTokensDetails
	return adapters.Usage{
		Model:             model,
		InputTokens:       max(r.Usage.PromptTokens-details.AudioTokens-details.CachedTokens, 0),
		CachedInputTokens: details.CachedTokens,
		AudioInputTokens:  details.AudioTokens,
		OutputTokens:      r.Usage.CompletionTokens,
	}
}

// arguments is the forced call's arguments; a refusal, a reply cut short or
// one without the call is a bad reply.
func (r chatReply) arguments(name string) (string, error) {
	if len(r.Choices) == 0 {
		return "", adapters.BadReply("chat completions", "no choice in the reply")
	}
	choice := r.Choices[0]
	switch {
	case choice.Message.Refusal != "":
		return "", fmt.Errorf("chat completions: %s: %w", choice.Message.Refusal, adapters.ErrRefused)
	case choice.FinishReason == "length":
		return "", fmt.Errorf("chat completions: %w", adapters.ErrTruncated)
	}
	for _, call := range choice.Message.ToolCalls {
		if call.Function.Name == name {
			return call.Function.Arguments, nil
		}
	}
	return "", adapters.BadReply("chat completions", "the reply made no call of "+name)
}
