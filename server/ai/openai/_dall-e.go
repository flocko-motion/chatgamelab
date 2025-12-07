package gpt

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"time"
	"webapp-server/obj"
)

func GenerateImage(ctx context.Context, apiKey string, prompt string) (image []byte, httpErr *obj.HTTPError) {
	client := newClient(apiKey)

	timeStart := time.Now()
	respUrl, err := client.CreateImage(
		ctx,
		openai.ImageRequest{
			Prompt:         prompt,
			Size:           openai.CreateImageSize256x256,
			ResponseFormat: openai.CreateImageResponseFormatB64JSON,
			N:              1,
		},
	)
	if err != nil {
		return nil, &obj.HTTPError{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("Image creation error: %v\n", err),
		}
	}
	log.Printf("Image created in %v\n", time.Since(timeStart))
	data, err := base64.StdEncoding.DecodeString(respUrl.Data[0].B64JSON)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed decoding generated image: " + err.Error()}
	}
	return data, nil
}
