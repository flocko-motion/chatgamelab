package mistral

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"engine/internal/adapters"
)

// imageInstructions keeps the model from answering in words instead of a picture.
const imageInstructions = "Generate the requested image. Do not add any text commentary."

// Image generates one picture through the image_generation tool. The tool
// produces a file, which a second call downloads.
type Image struct {
	keys  adapters.KeyFunc
	model string
	api   client
}

func NewImage(keys adapters.KeyFunc, model string) *Image {
	return &Image{
		keys:  keys,
		model: model,
		api:   client{base: baseURL, http: &http.Client{Timeout: 120 * time.Second}},
	}
}

func (i *Image) Generate(ctx context.Context, ask adapters.ImageRequest) ([]byte, adapters.Usage, error) {
	used := adapters.Usage{Model: i.model}

	key, err := i.keys(ctx)
	if err != nil {
		return nil, used, fmt.Errorf("resolve api key: %w", err)
	}

	out, err := i.api.converse(ctx, key, start{
		Model:        i.model,
		Instructions: imageInstructions,
		Inputs:       []input{{Role: "user", Content: ask.Prompt}},
		Tools:        []tool{{Type: "image_generation"}},
	})
	if err != nil {
		return nil, used, err
	}
	used.InputTokens = out.Usage.PromptTokens
	used.OutputTokens = out.Usage.CompletionTokens

	id := out.fileID()
	if id == "" {
		return nil, used, fmt.Errorf("mistral image: no file in response")
	}
	used.Images = 1

	image, err := i.api.file(ctx, key, id)
	if err != nil {
		return nil, used, err
	}
	if len(image) == 0 {
		return nil, used, fmt.Errorf("mistral image: empty file %s", id)
	}
	return image, used, nil
}
