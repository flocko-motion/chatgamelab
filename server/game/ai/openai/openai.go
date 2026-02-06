package openai

import (
	"bufio"
	"bytes"
	"cgl/apiclient"
	"cgl/functional"
	"cgl/game/imagecache"
	"cgl/game/stream"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type OpenAiPlatform struct{}

const (
	openaiBaseURL     = "https://api.openai.com/v1"
	responsesEndpoint = "/responses"
	modelsEndpoint    = "/models"
	imageGenEndpoint  = "/images/generations"
	defaultModel      = "gpt-5.2"
)

// extractImageErrorCode extracts an error code from OpenAI image generation errors
func extractImageErrorCode(err error) string {
	if err == nil {
		return ""
	}
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "invalid_api_key"):
		return obj.ErrCodeInvalidApiKey
	case strings.Contains(errStr, "billing_not_active"):
		return obj.ErrCodeBillingNotActive
	case strings.Contains(errStr, "organization_verification_required"),
		strings.Contains(errStr, "organization must be verified"),
		strings.Contains(errStr, "must be verified"):
		return obj.ErrCodeOrgVerificationRequired
	case strings.Contains(errStr, "rate_limit") || strings.Contains(errStr, "rate limit"):
		return obj.ErrCodeRateLimitExceeded
	case strings.Contains(errStr, "insufficient_quota") || strings.Contains(errStr, "quota"):
		return obj.ErrCodeInsufficientQuota
	case strings.Contains(errStr, "content_policy") || strings.Contains(errStr, "content_filter"):
		return obj.ErrCodeContentFiltered
	default:
		return obj.ErrCodeAiError
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
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
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

// newApi creates a new API client for OpenAI with the given API key
func (p *OpenAiPlatform) newApi(apiKey string) *apiclient.Client {
	return apiclient.NewApi(openaiBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
}

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

	// Convert to minimal AI message structure (exclude internal fields like ID, Meta, etc.)
	actionForAI := obj.GameSessionMessageAi{
		Type:         action.Type,
		Message:      action.Message,
		StatusFields: action.StatusFields,
		ImagePrompt:  action.ImagePrompt,
	}

	// Serialize the player action as JSON input
	actionInput, err := json.Marshal(actionForAI)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Build the request
	req := ResponsesAPIRequest{
		Model:           session.AiModel,
		Input:           string(actionInput),
		Store:           true,
		MaxOutputTokens: 5000,
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
		log.Error("OpenAI API call failed",
			"error", err,
			"session_id", session.ID,
			"model", req.Model,
		)
		return fmt.Errorf("OpenAI API error: %w", err)
	}
	log.Debug("OpenAI API call completed", "response_id", apiResponse.ID, "status", apiResponse.Status)

	if apiResponse.Status != "completed" {
		incompleteReason := ""
		if apiResponse.IncompleteDetails != nil {
			incompleteReason = apiResponse.IncompleteDetails.Reason
		}

		// Build a user-friendly error message
		var errMsg string
		switch {
		case apiResponse.Error != nil:
			errMsg = apiResponse.Error.Message
		case incompleteReason != "":
			errMsg = fmt.Sprintf("the AI response was incomplete (reason: %s)", incompleteReason)
		case apiResponse.Status == "failed":
			errMsg = "the AI failed to generate a response"
		default:
			errMsg = fmt.Sprintf("the AI returned an unexpected status: %s", apiResponse.Status)
		}

		log.Error("OpenAI response not completed",
			"status", apiResponse.Status,
			"error_message", errMsg,
			"incomplete_reason", incompleteReason,
			"response_id", apiResponse.ID,
			"session_id", session.ID,
			"model", req.Model,
		)
		return fmt.Errorf("%s", errMsg)
	}

	// Extract the text response
	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return fmt.Errorf("no text response from OpenAI")
	}

	response.ResponseRaw = &responseText

	// Parse the structured response into the pre-created message
	log.Debug("parsing OpenAI response", "response_length", len(responseText))
	if err := json.Unmarshal([]byte(responseText), response); err != nil {
		log.Debug("failed to parse game response", "error", err, "response_text", responseText[:min(200, len(responseText))])
		return fmt.Errorf("failed to parse game response: %w", err)
	}

	// Update model session with new response ID
	modelSession.ResponseID = apiResponse.ID
	response.URLAnalytics = functional.Ptr("https://platform.openai.com/logs/" + apiResponse.ID)
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
	client := apiclient.NewApi(openaiBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})

	var apiResp ResponsesAPIResponse
	if err := client.PostJson(ctx, responsesEndpoint, req, &apiResp); err != nil {
		return nil, err
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
		MaxOutputTokens:    5000,
		PreviousResponseID: modelSession.ResponseID,
	}

	// Make streaming API call
	log.Debug("calling OpenAI streaming API for story expansion")
	fullText, newResponseID, err := callStreamingResponsesAPI(ctx, session.ApiKey.Key, req, responseStream)
	if err != nil {
		// For story expansion, partial text is still usable — don't fail if we got some output
		if len(fullText) > 0 {
			log.Warn("OpenAI streaming API incomplete, using partial text",
				"error", err,
				"session_id", session.ID,
				"model", session.AiModel,
				"text_length", len(fullText),
			)
		} else {
			log.Error("OpenAI streaming API failed",
				"error", err,
				"session_id", session.ID,
				"model", session.AiModel,
			)
			return fmt.Errorf("OpenAI streaming API error: %w", err)
		}
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

// GenerateImage generates an image from the imagePrompt using the image cache
func (p *OpenAiPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	log.Debug("OpenAI GenerateImage starting", "session_id", session.ID, "message_id", response.ID)

	if session.ApiKey == nil {
		return fmt.Errorf("session has no API key")
	}

	// Note: imagePrompt check is done in game_logic.go before calling this function
	// to avoid race conditions with the shared response pointer
	log.Debug("generating image", "prompt_length", len(*response.ImagePrompt), "style", session.ImageStyle)

	// Initialize cache entry with image saver for persistence
	cache := imagecache.Get()
	cache.Create(response.ID, imagecache.ImageSaverFunc(responseStream.ImageSaver))

	// Build image generation request - writes to cache for polling
	imageData, err := callImageGenerationAPI(ctx, session.ApiKey.Key, *response.ImagePrompt, session.ImageStyle, response.ID, responseStream)
	if err != nil {
		log.Error("image generation failed",
			"error", err,
			"session_id", session.ID,
			"style", session.ImageStyle,
		)
		errorCode := extractImageErrorCode(err)
		cache.SetError(response.ID, errorCode, err.Error()) // Mark as failed so frontend knows to stop polling

		return fmt.Errorf("OpenAI image generation error: %w", err)
	}
	log.Debug("image generation completed", "image_size", len(imageData))

	// Update response with final image
	response.Image = imageData

	return nil
}

// callStreamingResponsesAPI makes a streaming call to the Responses API
// Note: Uses direct HTTP instead of apiclient because it requires SSE (Server-Sent Events) streaming
// with line-by-line processing and keeping the connection open for incremental responses
func callStreamingResponsesAPI(ctx context.Context, apiKey string, req ResponsesAPIRequest, responseStream *stream.Stream) (fullText string, responseID string, err error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	responsesURL := openaiBaseURL + responsesEndpoint
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
// Note: Uses direct HTTP instead of apiclient because it requires SSE streaming with custom buffer sizes
// for large base64-encoded image data and incremental partial image previews
func callImageGenerationAPI(ctx context.Context, apiKey string, prompt string, style string, messageID uuid.UUID, responseStream *stream.Stream) ([]byte, error) {
	imageGenURL := openaiBaseURL + imageGenEndpoint

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

// Translate translates language files to a target language using OpenAI API
func (p *OpenAiPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, error) {
	return "", fmt.Errorf("translation not yet implemented for openai - use mistral instead")
}

// ListModels retrieves all available models from OpenAI API
func (p *OpenAiPlatform) ListModels(ctx context.Context, apiKey string) ([]obj.AiModel, error) {
	client := p.newApi(apiKey)

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := client.GetJson(ctx, modelsEndpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	// Check if we should show all models without filtering
	showAll := false
	if val := ctx.Value("showAll"); val != nil {
		showAll, _ = val.(bool)
	}

	models := make([]obj.AiModel, 0, len(response.Data))
	for _, model := range response.Data {
		// Apply filters only if not showing all
		if !showAll {
			// Skip non-chat models
			if isIrrelevantModel(model.ID) {
				continue
			}

			// Skip preview models
			if strings.Contains(model.ID, "-preview") {
				continue
			}

			// Skip dated models (ending with YYYY-MM-DD pattern or 4-digit versions)
			if isDatedModel(model.ID) {
				continue
			}
		}

		models = append(models, obj.AiModel{
			ID:          model.ID,
			Name:        model.ID,
			Description: fmt.Sprintf("OpenAI model: %s", model.ID),
		})
	}

	// Sort alphabetically by model ID
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	return models, nil
}

// isIrrelevantModel checks if a model is known to NOT support chat completions
func isIrrelevantModel(modelID string) bool {
	// List of known non-chatmodel prefixes
	nonChatPrefixes := []string{
		"sora-",
		"tts-",
		"whisper-",
		"text-embedding-",
		"omni-moderation-",
		"codex-",
	}

	for _, prefix := range nonChatPrefixes {
		if strings.HasPrefix(modelID, prefix) {
			return true
		}
	}

	return false
}

// isDatedModel checks if a model ID ends with a date or version pattern
func isDatedModel(modelID string) bool {
	parts := strings.Split(modelID, "-")
	if len(parts) < 2 {
		return false
	}

	lastPart := parts[len(parts)-1]

	// Check if ends with 4 digits (e.g., -1106, -0914)
	if len(lastPart) == 4 {
		for _, ch := range lastPart {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true
	}

	// Check if model ends with -YYYY-MM-DD pattern
	if len(parts) < 3 {
		return false
	}

	// Get last 3 parts
	lastThree := parts[len(parts)-3:]

	// Check if they match YYYY-MM-DD pattern
	if len(lastThree[0]) == 4 && len(lastThree[1]) == 2 && len(lastThree[2]) == 2 {
		// Verify they're all numeric
		for _, part := range lastThree {
			for _, ch := range part {
				if ch < '0' || ch > '9' {
					return false
				}
			}
		}
		return true
	}

	return false
}

// GenerateTheme generates a visual theme JSON for the game player UI
func (p *OpenAiPlatform) GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, error) {
	log.Debug("OpenAI GenerateTheme starting", "session_id", session.ID)

	if session.ApiKey == nil {
		return "", fmt.Errorf("session has no API key")
	}

	api := p.newApi(session.ApiKey.Key)

	// Use a simple request for theme generation - no need for structured output
	reqBody := ResponsesAPIRequest{
		Model:        session.AiModel,
		Instructions: systemPrompt,
		Input:        userPrompt,
		Store:        false, // Don't store theme generation in conversation history
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := api.Post(ctx, responsesEndpoint, reqBytes)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp ResponsesAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	// Extract text from response
	for _, output := range apiResp.Output {
		if output.Type == "message" {
			for _, content := range output.Content {
				if content.Type == "output_text" || content.Type == "text" {
					log.Debug("OpenAI GenerateTheme completed", "response_length", len(content.Text))
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no text content in response")
}
