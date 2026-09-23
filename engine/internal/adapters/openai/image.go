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
	keys    adapters.KeyFunc
	model   string
	quality string
	size    string
	client  *http.Client
}

func NewImage(keys adapters.KeyFunc, model, quality, size string) *Image {
	return &Image{
		keys:    keys,
		model:   model,
		quality: quality,
		size:    size,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (i *Image) Generate(ctx context.Context, ask adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	used := adapters.Usage{Model: i.model}

	key, err := i.keys(ctx)
	if err != nil {
		return nil, used, fmt.Errorf("resolve api key: %w", err)
	}

	// Anything left empty is left out, so the provider's own default applies
	// rather than this adapter inventing one.
	request := map[string]any{
		"model":  i.model,
		"prompt": ask.Prompt,
		"n":      1,
	}
	if i.size != "" {
		request["size"] = i.size
	}
	if i.quality != "" {
		request["quality"] = i.quality
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, used, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imagesURL, bytes.NewReader(body))
	if err != nil {
		return nil, used, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, used, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, used, fmt.Errorf("images api: %s", resp.Status)
	}

	var out struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, used, err
	}
	used.InputTokens = out.Usage.InputTokens
	used.OutputTokens = out.Usage.OutputTokens
	used.Images = int64(len(out.Data))

	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		return nil, used, fmt.Errorf("images api: empty response")
	}
	image, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	return image, used, err
}
