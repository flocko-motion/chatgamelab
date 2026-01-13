package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cgl/game/stream"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"
)

const (
	responsesURL = "https://api.openai.com/v1/responses"
	defaultModel = "gpt-4o-mini"
)

type OpenAiPlatform struct{}

func (p *OpenAiPlatform) GetPlatformInfo() obj.AiPlatform {
	return obj.AiPlatform{
		ID:   "openai",
		Name: "OpenAI",
		Models: []obj.AiModel{
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Fast and cost-effective for most tasks"},
			{ID: "gpt-4o", Name: "GPT-4o", Description: "Most capable model for complex tasks"},
			{ID: "gpt-4.1-nano", Name: "GPT-4.1 Nano", Description: "Cheapest option for simple tasks"},
		},
	}
}

// ModelSession stores the OpenAI response ID for conversation continuity
type ModelSession struct {
	ResponseID string `json:"responseId"`
}

// ResponsesAPIRequest is the request body for the Responses API
type ResponsesAPIRequest struct {
	Model              string      `json:"model"`
	Input              string      `json:"input"`
	Instructions       string      `json:"instructions,omitempty"`
	PreviousResponseID string      `json:"previous_response_id,omitempty"`
	Store              bool        `json:"store"`
	Stream             bool        `json:"stream,omitempty"`
	MaxOutputTokens    int         `json:"max_output_tokens,omitempty"`
	Text               *TextConfig `json:"text,omitempty"`
}

type TextConfig struct {
	Format FormatConfig `json:"format"`
}

type FormatConfig struct {
	Type   string      `json:"type"`
	Name   string      `json:"name,omitempty"`
	Schema interface{} `json:"schema,omitempty"`
	Strict bool        `json:"strict,omitempty"`
}

// ResponsesAPIResponse is the response from the Responses API
type ResponsesAPIResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	Output []OutputItem `json:"output"`
}

type OutputItem struct {
	Type    string        `json:"type"`
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (p *OpenAiPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage) error {
	log.Debug("OpenAI ExecuteAction starting", "session_id", session.ID, "action_type", action.Type, "model", session.AiModel)

	if session.ApiKey == nil {
		return fmt.Errorf("session has no API key")
	}

	// Parse the model session
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		log.Debug("failed to parse model session", "error", err)
		return fmt.Errorf("failed to parse model session: %w", err)
	}

	// Serialize the player action as JSON input
	actionInput, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Build the request
	req := ResponsesAPIRequest{
		Model:           session.AiModel,
		Input:           string(actionInput),
		Store:           true,
		MaxOutputTokens: 500, // Keep responses concise
		Text: &TextConfig{
			Format: FormatConfig{
				Type:   "json_schema",
				Name:   "game_response",
				Schema: obj.GameResponseSchema,
				Strict: true,
			},
		},
	}

	// System messages become instructions, otherwise use previous_response_id for continuity
	if action.Type == obj.GameSessionMessageTypeSystem {
		req.Instructions = action.Message
		req.Input = lang.T("aiMessageStart")
	} else if modelSession.ResponseID != "" {
		req.PreviousResponseID = modelSession.ResponseID
	}

	responseStream := stream.Get().Lookup(response.ID)
	if responseStream == nil {
		return fmt.Errorf("stream not found for message %s", response.ID)
	}

	// Make the API call
	log.Debug("calling OpenAI Responses API", "model", req.Model, "has_previous_response", req.PreviousResponseID != "")
	apiResponse, err := callResponsesAPI(ctx, session.ApiKey.Key, req)
	if err != nil {
		log.Debug("OpenAI API call failed", "error", err)
		return fmt.Errorf("OpenAI API error: %w", err)
	}
	log.Debug("OpenAI API call completed", "response_id", apiResponse.ID, "status", apiResponse.Status)

	if apiResponse.Status != "completed" {
		errMsg := "unknown error"
		if apiResponse.Error != nil {
			errMsg = apiResponse.Error.Message
		}
		return fmt.Errorf("response failed: %s", errMsg)
	}

	// Extract the text response
	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return fmt.Errorf("no text response from OpenAI")
	}

	// Parse the structured response into the pre-created message
	log.Debug("parsing OpenAI response", "response_length", len(responseText))
	if err := json.Unmarshal([]byte(responseText), response); err != nil {
		log.Debug("failed to parse game response", "error", err, "response_text", responseText[:min(200, len(responseText))])
		return fmt.Errorf("failed to parse game response: %w", err)
	}

	// Update model session with new response ID
	modelSession.ResponseID = apiResponse.ID
	sessionJSON, err := json.Marshal(modelSession)
	if err != nil {
		return fmt.Errorf("failed to marshal model session: %w", err)
	}
	session.AiSession = string(sessionJSON)

	// Set fields that come from the session, not from GPT
	response.GameSessionID = session.ID
	response.Type = obj.GameSessionMessageTypeGame

	return nil
}

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

func callResponsesAPI(ctx context.Context, apiKey string, req ResponsesAPIRequest) (*ResponsesAPIResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", responsesURL, bytes.NewReader(reqBody))
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ResponsesAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}

// ExpandStory expands the plot outline to full narrative text using streaming
func (p *OpenAiPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	log.Debug("OpenAI ExpandStory starting", "session_id", session.ID, "message_id", response.ID)

	if session.ApiKey == nil {
		return fmt.Errorf("session has no API key")
	}

	// Parse the model session to get previous response ID
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		return fmt.Errorf("failed to parse model session: %w", err)
	}

	// Build streaming request - plain text, no JSON schema
	req := ResponsesAPIRequest{
		Model:              session.AiModel,
		Input:              lang.T("aiExpandPlotOutline"),
		Store:              true,
		Stream:             true,
		MaxOutputTokens:    400, // Keep expanded text concise
		PreviousResponseID: modelSession.ResponseID,
	}

	// Make streaming API call
	log.Debug("calling OpenAI streaming API for story expansion")
	fullText, newResponseID, err := callStreamingResponsesAPI(ctx, session.ApiKey.Key, req, responseStream)
	if err != nil {
		log.Debug("OpenAI streaming API failed", "error", err)
		return fmt.Errorf("OpenAI streaming API error: %w", err)
	}
	log.Debug("story expansion completed", "text_length", len(fullText), "new_response_id", newResponseID)

	// Update response with full text
	response.Message = fullText

	// Update model session with new response ID
	modelSession.ResponseID = newResponseID
	sessionJSON, err := json.Marshal(modelSession)
	if err != nil {
		return fmt.Errorf("failed to marshal model session: %w", err)
	}
	session.AiSession = string(sessionJSON)

	return nil
}

// GenerateImage generates an image from the imagePrompt using streaming
func (p *OpenAiPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	log.Debug("OpenAI GenerateImage starting", "session_id", session.ID, "message_id", response.ID)

	if session.ApiKey == nil {
		return fmt.Errorf("session has no API key")
	}

	if response.ImagePrompt == nil || *response.ImagePrompt == "" {
		log.Debug("no image prompt, skipping image generation")
		return nil // No image to generate
	}
	log.Debug("generating image", "prompt_length", len(*response.ImagePrompt), "style", session.ImageStyle)

	// Build image generation request with streaming
	imageData, err := callImageGenerationAPI(ctx, session.ApiKey.Key, *response.ImagePrompt, session.ImageStyle, responseStream)
	if err != nil {
		log.Debug("image generation failed", "error", err)
		return fmt.Errorf("OpenAI image generation error: %w", err)
	}
	log.Debug("image generation completed", "image_size", len(imageData))

	// Update response with final image
	response.Image = imageData

	return nil
}

// callStreamingResponsesAPI makes a streaming call to the Responses API
func callStreamingResponsesAPI(ctx context.Context, apiKey string, req ResponsesAPIRequest, responseStream *stream.Stream) (fullText string, responseID string, err error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", responsesURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
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

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "response.output_text.delta":
			if delta, ok := event["delta"].(string); ok {
				textBuilder.WriteString(delta)
				responseStream.SendText(delta, false)
			}
		case "response.completed":
			if respObj, ok := event["response"].(map[string]interface{}); ok {
				responseID, _ = respObj["id"].(string)
			}
		}
	}

	// Signal text streaming complete
	responseStream.SendText("", true)
	return textBuilder.String(), responseID, nil
}

// callImageGenerationAPI generates an image with streaming partial images
func callImageGenerationAPI(ctx context.Context, apiKey string, prompt string, style string, responseStream *stream.Stream) ([]byte, error) {
	const imageGenURL = "https://api.openai.com/v1/images/generations"

	// Note: style parameter is only supported for dall-e-3, not gpt-image-1
	// For gpt-image-1, we include the style in the prompt instead
	fullPrompt := prompt
	if style != "" {
		fullPrompt = fmt.Sprintf("%s. Style: %s", prompt, style)
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

			switch eventType {
			case "image_generation.partial_image":
				responseStream.SendImage(imageData, false)
			case "image_generation.completed":
				finalImageData = imageData
				responseStream.SendImage(imageData, true)
			}
		}
	}

	return finalImageData, nil
}
