package mistral

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"engine/internal/adapters"
)

// Hearing hears a clip of the player's speech on Voxtral, through Chat
// Completions: the only Mistral API that takes audio with instructions. The
// clip goes as WAV, and the reply is JSON of the schema in strict mode, which
// Voxtral keeps with audio in: every reply fit the schema when cmd/probe
// tried it on 21 November 2026.
//
// Voxtral bills the audio it hears per minute, so the clip's length is the
// usage's AudioSeconds, beside the text tokens.
type Hearing struct {
	keys  adapters.KeyFunc
	model string
	api   client
}

func NewHearing(keys adapters.KeyFunc, model string) *Hearing {
	return &Hearing{keys: keys, model: model, api: client{base: baseURL, http: &http.Client{Timeout: 30 * time.Second}}}
}

func (h *Hearing) Hear(ctx context.Context, system string, clip adapters.Clip, schema adapters.Schema) (string, adapters.Usage, error) {
	used := adapters.Usage{Model: h.model}
	key, err := h.keys(ctx)
	if err != nil {
		return "", used, fmt.Errorf("resolve api key: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"model": h.model,
		"messages": []map[string]any{
			{"role": "system", "content": system},
			{"role": "user", "content": []map[string]any{{
				"type":        "input_audio",
				"input_audio": base64.StdEncoding.EncodeToString(clip.WAV()),
			}}},
		},
		"response_format": responseFormat(&schema),
	})
	if err != nil {
		return "", used, err
	}
	resp, err := h.api.do(ctx, key, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return "", used, err
	}
	defer resp.Body.Close()

	var out struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", used, adapters.Transient("mistral /chat/completions", err)
	}
	used.InputTokens = out.Usage.PromptTokens
	used.OutputTokens = out.Usage.CompletionTokens
	used.AudioSeconds = clip.Seconds()

	if len(out.Choices) == 0 {
		return "", used, adapters.BadReply("mistral /chat/completions", "no choice in the reply")
	}
	choice := out.Choices[0]
	if choice.FinishReason == "length" {
		return "", used, fmt.Errorf("mistral /chat/completions: %w", adapters.ErrTruncated)
	}
	return content(choice.Message.Content), used, nil
}

// content is a message's text, in either content shape: a string, or chunks.
func content(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var text string
	for _, c := range (reply{}).chunks(raw) {
		if c.Type == "text" {
			text += c.Text
		}
	}
	return text
}
