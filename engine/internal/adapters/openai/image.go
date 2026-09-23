package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"engine/internal/adapters"
)

// Image generates one picture. Like the other adapters here it follows the
// documented request shape but has not been run against the API.
type Image struct {
	keys   adapters.KeyFunc
	model  string
	client *http.Client
}

func NewImage(keys adapters.KeyFunc, model string) *Image {
	return &Image{
		keys:   keys,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (i *Image) Generate(ctx context.Context, prompt string) ([]byte, error) {
	key, err := i.keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve api key: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"model":  i.model,
		"prompt": prompt,
		"n":      1,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imagesURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("images api: %s", resp.Status)
	}

	var out struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		return nil, fmt.Errorf("images api: empty response")
	}
	return base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
}
