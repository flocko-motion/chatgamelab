package openai

import (
	"bufio"
	"bytes"
	"cgl/apiclient"
	"cgl/game/imagecache"
	"cgl/game/stream"
	"cgl/game/templates"
	"cgl/log"
	"cgl/obj"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// extractResponseText extracts the text content from an OpenAI Responses API response
func extractResponseText(apiResponse *ResponsesAPIResponse) string {
	for _, output := range apiResponse.Output {
		if output.Type == "message" && output.Role == "assistant" {
			for _, content := range output.Content {
				if content.Type == "output_text" {
					return content.Text
				}
			}
		}
	}
	return ""
}

func callResponsesAPI(ctx context.Context, apiKey string, req ResponsesAPIRequest) (*ResponsesAPIResponse, obj.TokenUsage, error) {
	client := apiclient.NewApi(openaiBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})

	var apiResp ResponsesAPIResponse
	if err := client.PostJson(ctx, responsesEndpoint, req, &apiResp); err != nil {
		return nil, obj.TokenUsage{}, err
	}

	usage := apiResp.Usage.toTokenUsage()
	log.Debug("API token usage", "input_tokens", usage.InputTokens, "output_tokens", usage.OutputTokens, "total_tokens", usage.TotalTokens)
	return &apiResp, usage, nil
}

// callStreamingResponsesAPI makes a streaming call to the Responses API
// Note: Uses direct HTTP instead of apiclient because it requires SSE (Server-Sent Events) streaming
// with line-by-line processing and keeping the connection open for incremental responses
func callStreamingResponsesAPI(ctx context.Context, apiKey string, req ResponsesAPIRequest, responseStream *stream.Stream) (fullText string, responseID string, usage obj.TokenUsage, err error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", "", obj.TokenUsage{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	responsesURL := openaiBaseURL + responsesEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, "POST", responsesURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", "", obj.TokenUsage{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", "", obj.TokenUsage{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", obj.TokenUsage{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	var textBuilder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event sseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "response.output_text.delta":
			var delta sseTextDelta
			if json.Unmarshal([]byte(data), &delta) == nil {
				textBuilder.WriteString(delta.Delta)
				responseStream.SendText(delta.Delta, false)
			}
		case "response.completed":
			var completed sseResponseCompleted
			if json.Unmarshal([]byte(data), &completed) == nil {
				responseID = completed.Response.ID
				usage = completed.Response.Usage.toTokenUsage()
			}
		}
	}

	log.Debug("streaming API token usage", "input_tokens", usage.InputTokens, "output_tokens", usage.OutputTokens, "total_tokens", usage.TotalTokens)
	// Signal text streaming complete
	responseStream.SendText("", true)
	return textBuilder.String(), responseID, usage, nil
}

// callImageGenerationAPI generates an image with streaming partial images
// Note: Uses direct HTTP instead of apiclient because it requires SSE streaming with custom buffer sizes
// for large base64-encoded image data and incremental partial image previews
func callImageGenerationAPI(ctx context.Context, apiKey string, prompt string, style string, messageID uuid.UUID, responseStream *stream.Stream) ([]byte, error) {
	imageGenURL := openaiBaseURL + imageGenEndpoint

	// Note: style parameter is only supported for dall-e-3, not gpt-image-1
	// For gpt-image-1, we include the style in the prompt instead
	fullPrompt := prompt + templates.ImagePromptSuffix
	if style != "" {
		fullPrompt = fmt.Sprintf("%s Style: %s", fullPrompt, style)
	}

	reqBody := map[string]interface{}{
		"model":          "gpt-image-1",
		"prompt":         fullPrompt,
		"n":              1,
		"size":           "1024x1024",
		"quality":        "low",
		"output_format":  "png",
		"stream":         true,
		"partial_images": 3, // Get previews of the image generation process - each preview is sent as a full png file
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", imageGenURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream for image events
	var finalImageData []byte
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for base64 image data
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		b64Json, _ := event["b64_json"].(string)

		if b64Json != "" {
			imageData, err := base64.StdEncoding.DecodeString(b64Json)
			if err != nil {
				continue
			}

			// Update cache for polling-based frontend
			cache := imagecache.Get()

			switch eventType {
			case "image_generation.partial_image":
				cache.Update(messageID, imageData, false)
				responseStream.SendImage(imageData, false)
			case "image_generation.completed":
				finalImageData = imageData
				cache.Update(messageID, imageData, true) // This also persists to DB
				responseStream.SendImage(imageData, true)
			}
		}
	}

	return finalImageData, nil
}
