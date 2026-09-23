package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"engine/internal/adapters"
)

// Tool is one single-shot text call. The observer's classifier runs on it, and
// so would a rephrase or a translation — the role is the prompt, not the type.
type Tool struct {
	keys   adapters.KeyFunc
	model  string
	client *http.Client
}

func NewTool(keys adapters.KeyFunc, model string) *Tool {
	return &Tool{
		keys:  keys,
		model: model,
		// The observer runs concurrently with speech, so its latency is hidden
		// only while it stays well under one spoken utterance.
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *Tool) Query(ctx context.Context, system, user string) (string, adapters.Usage, error) {
	var used adapters.Usage

	key, err := t.keys(ctx)
	if err != nil {
		return "", used, fmt.Errorf("resolve api key: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"model":        t.model,
		"instructions": system,
		"input":        user,
	})
	if err != nil {
		return "", used, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, bytes.NewReader(body))
	if err != nil {
		return "", used, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", used, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", used, fmt.Errorf("responses api: %s", resp.Status)
	}

	var out struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens        int64 `json:"input_tokens"`
			OutputTokens       int64 `json:"output_tokens"`
			InputTokensDetails struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", used, err
	}

	cached := out.Usage.InputTokensDetails.CachedTokens
	used = adapters.Usage{
		Model: t.model,
		// Cached input is priced separately, so it is reported separately
		// rather than folded into the input total.
		InputTokens:       out.Usage.InputTokens - cached,
		CachedInputTokens: cached,
		OutputTokens:      out.Usage.OutputTokens,
	}

	for _, o := range out.Output {
		for _, c := range o.Content {
			if c.Text != "" {
				return c.Text, used, nil
			}
		}
	}
	return "", used, fmt.Errorf("responses api: empty output")
}
