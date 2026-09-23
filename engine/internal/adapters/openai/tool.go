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

func (t *Tool) Query(ctx context.Context, system, user string) (string, error) {
	key, err := t.keys(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve api key: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"model":        t.model,
		"instructions": system,
		"input":        user,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("responses api: %s", resp.Status)
	}

	var out struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	for _, o := range out.Output {
		for _, c := range o.Content {
			if c.Text != "" {
				return c.Text, nil
			}
		}
	}
	return "", fmt.Errorf("responses api: empty output")
}
