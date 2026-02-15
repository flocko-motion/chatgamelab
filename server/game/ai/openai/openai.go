package openai

import (
	"bytes"
	"cgl/functional"
	"cgl/game/imagecache"
	"cgl/game/status"
	"cgl/game/stream"
	"cgl/game/templates"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
)

type OpenAiPlatform struct{}

func (p *OpenAiPlatform) GetPlatformInfo() obj.AiPlatform {
	return obj.AiPlatform{
		ID:   "openai",
		Name: "OpenAI",
		Models: []obj.AiModel{
			{
				ID:               obj.AiModelMax,
				Name:             "GPT-5.2 + Audio",
				Model:            "gpt-5.2",
				ImageModel:       "gpt-image-1.5",
				ImageQuality:     "medium",
				Description:      "Highest + Audio",
				SupportsImage:    true,
				SupportsAudioIn:  true,
				SupportsAudioOut: true,
			}, {
				ID:              obj.AiModelPremium,
				Name:            "GPT-5.2",
				Model:           "gpt-5.2",
				ImageModel:      "gpt-image-1.5",
				ImageQuality:    "medium",
				Description:     "Premium",
				SupportsImage:   true,
				SupportsAudioIn: true,
			}, {
				ID:            obj.AiModelBalanced,
				Name:          "GPT-5.1",
				Model:         "gpt-5.1",
				ImageModel:    "gpt-image-1.5",
				ImageQuality:  "low",
				Description:   "Balanced",
				SupportsImage: true,
			}, {
				ID:            obj.AiModelEconomy,
				Name:          "GPT-5 Mini",
				Model:         "gpt-5-mini",
				ImageModel:    "gpt-image-1-mini",
				ImageQuality:  "low",
				Description:   "Economy",
				SupportsImage: true,
			},
		},
	}
}

func (p *OpenAiPlatform) ResolveModelInfo(tierID string) *obj.AiModel {
	info := p.GetPlatformInfo()
	return info.ResolveModelWithDowngrade(tierID)
}

func (p *OpenAiPlatform) ResolveModel(tierID string) string {
	if m := p.ResolveModelInfo(tierID); m != nil {
		return m.Model
	}
	// fallback: medium tier
	return p.GetPlatformInfo().Models[1].Model
}

func (p *OpenAiPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage, gameSchema map[string]interface{}) (obj.TokenUsage, error) {
	model := p.ResolveModel(session.AiModel)

	if session.ApiKey == nil {
		return obj.TokenUsage{}, obj.ErrInvalidApiKey("session has no API key")
	}

	// Parse the model session
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		log.Warn("failed to parse model session", "error", err, "ai_session_raw", session.AiSession)
		return obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to parse model session", err)
	}

	// Serialize the player action as JSON input (minimal AI-facing structure)
	actionInput := action.ToAiJSON()

	// Build the request (Input is set below based on action type)
	req := ResponsesAPIRequest{
		Model:           model,
		Input:           []InputMessage{{Role: "user", Content: actionInput}},
		Store:           true,
		MaxOutputTokens: 5000,
		Text: &TextConfig{
			Format: FormatConfig{
				Type:   "json_schema",
				Name:   "game_response",
				Schema: gameSchema,
				Strict: true,
			},
		},
	}

	// System messages become instructions, otherwise use previous_response_id for continuity
	if action.Type == obj.GameSessionMessageTypeSystem {
		req.Instructions = action.Message
		req.Input = []InputMessage{{Role: "developer", Content: templates.PromptMessageStart}}
	} else if modelSession.ResponseID != "" {
		req.PreviousResponseID = modelSession.ResponseID
		// Inject developer reminder with every player action to reinforce brevity
		req.Input = []InputMessage{
			{Role: "developer", Content: templates.ReminderExecuteAction},
			{Role: "user", Content: actionInput},
		}
		// Set debug prompt showing full input sent to the AI
		response.PromptStatusUpdate = functional.Ptr(
			"[developer] " + templates.ReminderExecuteAction + "\n\n[user] " + actionInput)
	}

	responseStream := stream.Get().Lookup(response.ID)
	if responseStream == nil {
		return obj.TokenUsage{}, obj.ErrServerErrorf("stream not found for message %s", response.ID)
	}

	// Make the API call
	apiResponse, usage, err := callResponsesAPI(ctx, session.ApiKey.Key, req)
	if err != nil {
		log.Error("OpenAI API call failed",
			"error", err,
			"session_id", session.ID,
			"model", req.Model,
		)
		return obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "OpenAI API error", err)
	}

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
		return usage, obj.ErrAiError(errMsg)
	}

	// Extract the text response
	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return usage, obj.ErrAiError("no text response from OpenAI")
	}

	response.ResponseRaw = &responseText

	// Parse the AI response and convert to internal format
	if err := status.ParseGameResponse(responseText, session.StatusFields, action.StatusFields, response); err != nil {
		log.Error("failed to parse game response", "error", err, "response_text", responseText)
		return usage, err
	}

	// Update model session with new response ID
	modelSession.ResponseID = apiResponse.ID
	response.URLAnalytics = functional.Ptr("https://platform.openai.com/logs/" + apiResponse.ID)
	sessionJSON, err := json.Marshal(modelSession)
	if err != nil {
		return usage, obj.WrapError(obj.ErrCodeAiError, "failed to marshal model session", err)
	}
	session.AiSession = string(sessionJSON)

	// Set fields that come from the session, not from GPT
	response.GameSessionID = session.ID
	response.Type = obj.GameSessionMessageTypeGame

	return usage, nil
}

// ExpandStory expands the plot outline to full narrative text using streaming
func (p *OpenAiPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) (obj.TokenUsage, error) {
	log.Debug("OpenAI ExpandStory starting", "session_id", session.ID, "message_id", response.ID)

	if session.ApiKey == nil {
		return obj.TokenUsage{}, obj.ErrInvalidApiKey("session has no API key")
	}

	// Parse the model session to get previous response ID
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		return obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to parse model session", err)
	}

	// Build streaming request - plain text, no JSON schema
	// Use developer role for the narration instruction (it's a directive, not player input)
	model := p.ResolveModel(session.AiModel)
	req := ResponsesAPIRequest{
		Model: model,
		Input: []InputMessage{
			{Role: "developer", Content: templates.PromptNarratePlotOutline(session.Language)},
		},
		Store:              true,
		Stream:             true,
		MaxOutputTokens:    5000,
		PreviousResponseID: modelSession.ResponseID,
	}

	// Make streaming API call
	fullText, newResponseID, usage, err := callStreamingResponsesAPI(ctx, session.ApiKey.Key, req, responseStream)
	if err != nil {
		// For story expansion, partial text is still usable - don't fail if we got some output
		if len(fullText) > 0 {
			log.Warn("OpenAI streaming API incomplete, using partial text",
				"error", err,
				"session_id", session.ID,
				"model", model,
				"text_length", len(fullText),
			)
		} else {
			log.Error("OpenAI streaming API failed",
				"error", err,
				"session_id", session.ID,
				"model", model,
			)
			return usage, obj.WrapError(obj.ErrCodeAiError, "OpenAI streaming API error", err)
		}
	}

	// Update response with full text
	response.Message = fullText

	// Update model session with new response ID
	modelSession.ResponseID = newResponseID
	sessionJSON, err := json.Marshal(modelSession)
	if err != nil {
		return usage, obj.WrapError(obj.ErrCodeAiError, "failed to marshal model session", err)
	}
	session.AiSession = string(sessionJSON)

	return usage, nil
}

// GenerateImage generates an image from the imagePrompt using the image cache
func (p *OpenAiPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	log.Debug("OpenAI GenerateImage starting", "session_id", session.ID, "message_id", response.ID)

	if session.ApiKey == nil {
		return obj.ErrInvalidApiKey("session has no API key")
	}

	// Note: imagePrompt check is done in game_logic.go before calling this function
	// to avoid race conditions with the shared response pointer
	modelInfo := p.ResolveModelInfo(session.AiModel)
	imageModel := modelInfo.ImageModel
	imageQuality := modelInfo.ImageQuality
	if imageQuality == "" {
		imageQuality = "low"
	}

	// Initialize cache entry with image saver for persistence
	cache := imagecache.Get()
	cache.Create(response.ID, imagecache.ImageSaverFunc(responseStream.ImageSaver))

	// Build rich image prompt with full context (idea, scenario, current scene, visual, style)
	plotOutline := ""
	if response.Plot != nil {
		plotOutline = *response.Plot
	}
	scenarioForImage := functional.First(session.GameScenarioImagePrompt, session.GameScenario)
	fullPrompt := templates.BuildImagePrompt(session.GameDescription, scenarioForImage, plotOutline, functional.Deref(response.ImagePrompt, ""), session.ImageStyle)

	// Build image generation request - writes to cache for polling
	imageData, err := callImageGenerationAPI(ctx, session.ApiKey.Key, imageModel, imageQuality, fullPrompt, response.ID, responseStream)
	if err != nil {
		log.Error("image generation failed",
			"error", err,
			"session_id", session.ID,
			"style", session.ImageStyle,
		)
		errorCode := extractImageErrorCode(err)
		cache.SetError(response.ID, errorCode, err.Error()) // Mark as failed so frontend knows to stop polling

		return obj.WrapError(obj.ErrCodeAiError, "OpenAI image generation error", err)
	}

	// Update response with final image
	response.Image = imageData

	return nil
}

// GenerateAudio generates audio narration from text using OpenAI TTS API
func (p *OpenAiPlatform) GenerateAudio(ctx context.Context, session *obj.GameSession, text string, responseStream *stream.Stream) ([]byte, error) {
	log.Debug("OpenAI GenerateAudio starting", "session_id", session.ID, "text_length", len(text))

	if session.ApiKey == nil {
		return nil, obj.ErrInvalidApiKey("session has no API key")
	}

	audioData, err := callSpeechAPI(ctx, session.ApiKey.Key, text, responseStream)
	if err != nil {
		log.Error("TTS generation failed", "error", err, "session_id", session.ID)
		return nil, obj.WrapError(obj.ErrCodeAiError, "OpenAI TTS error", err)
	}

	return audioData, nil
}

// Translate translates language files to a target language using OpenAI API
func (p *OpenAiPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, obj.TokenUsage, error) {
	originals := ""
	for i, original := range input {
		originals += fmt.Sprintf("Original #%d: \n%s\n\n", i+1, original)
	}

	req := ResponsesAPIRequest{
		Model:        translateModel,
		Instructions: lang.TranslateInstruction,
		Input:        []InputMessage{{Role: "user", Content: fmt.Sprintf("Translate this JSON to %s:\n\n%s", lang.GetLanguageName(targetLang), originals)}},
		Store:        false,
		Text: &TextConfig{
			Format: FormatConfig{
				Type: "json_object",
			},
		},
	}

	apiResponse, usage, err := callResponsesAPI(ctx, apiKey, req)
	if err != nil {
		return "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to translate", err)
	}

	// Check for API-level errors (HTTP 200 but failed response)
	if apiResponse.Error != nil {
		return "", usage, obj.ErrAiErrorf("OpenAI error: %s", apiResponse.Error.Message)
	}
	if apiResponse.Status != "" && apiResponse.Status != "completed" {
		reason := apiResponse.Status
		if apiResponse.IncompleteDetails != nil {
			reason += ": " + apiResponse.IncompleteDetails.Reason
		}
		return "", usage, obj.ErrAiErrorf("OpenAI response not completed: %s", reason)
	}

	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return "", usage, obj.ErrAiErrorf("no text response from OpenAI (status: %s)", apiResponse.Status)
	}

	// Validate it's valid JSON
	var translated map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &translated); err != nil {
		return "", usage, obj.WrapError(obj.ErrCodeAiError, "failed to parse translated JSON", err)
	}

	return functional.MustAnyToJson(translated), usage, nil
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
		return nil, obj.WrapError(obj.ErrCodeAiError, "failed to get models", err)
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

// TranscribeAudio transcribes audio data to text
func (p *OpenAiPlatform) TranscribeAudio(ctx context.Context, apiKey string, audioData []byte, mimeType string) (string, error) {
	// Determine file extension from MIME type
	ext := ".webm"
	switch {
	case strings.Contains(mimeType, "ogg"):
		ext = ".ogg"
	case strings.Contains(mimeType, "mp3") || strings.Contains(mimeType, "mpeg"):
		ext = ".mp3"
	case strings.Contains(mimeType, "mp4"):
		ext = ".mp4"
	case strings.Contains(mimeType, "wav"):
		ext = ".wav"
	}

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("model", transcriptionModel)
	part, err := writer.CreateFormFile("file", "audio"+ext)
	if err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "failed to create form file", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "failed to write audio data", err)
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", openaiBaseURL+transcriptionEndpoint, &buf)
	if err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "failed to create request", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "transcription request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "failed to read transcription response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", obj.ErrAiErrorf("transcription API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "failed to parse transcription response", err)
	}

	return strings.TrimSpace(result.Text), nil
}

// ToolQuery sends a single text prompt and returns a text answer using a fast model.
func (p *OpenAiPlatform) ToolQuery(ctx context.Context, apiKey string, prompt string) (string, error) {
	req := ResponsesAPIRequest{
		Model: toolQueryModel,
		Input: []InputMessage{{Role: "user", Content: prompt}},
		Store: false,
	}

	apiResponse, _, err := callResponsesAPI(ctx, apiKey, req)
	if err != nil {
		return "", obj.WrapError(obj.ErrCodeAiError, "ToolQuery failed", err)
	}

	if apiResponse.Error != nil {
		return "", obj.ErrAiErrorf("ToolQuery error: %s", apiResponse.Error.Message)
	}

	text := extractResponseText(apiResponse)
	if text == "" {
		return "", obj.ErrAiError("ToolQuery: no text in response")
	}

	return text, nil
}

// GenerateTheme generates a visual theme JSON for the game player UI
func (p *OpenAiPlatform) GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, obj.TokenUsage, error) {
	if session.ApiKey == nil {
		return "", obj.TokenUsage{}, obj.ErrInvalidApiKey("session has no API key")
	}

	api := p.newApi(session.ApiKey.Key)

	// Use a simple request for theme generation - no need for structured output
	model := p.ResolveModel(session.AiModel)
	reqBody := ResponsesAPIRequest{
		Model:        model,
		Instructions: systemPrompt,
		Input:        []InputMessage{{Role: "user", Content: userPrompt}},
		Store:        false, // Don't store theme generation in conversation history
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to marshal request", err)
	}

	resp, err := api.Post(ctx, responsesEndpoint, reqBytes)
	if err != nil {
		return "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "API request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", obj.TokenUsage{}, obj.ErrAiErrorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp ResponsesAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to parse response", err)
	}

	usage := apiResp.Usage.toTokenUsage()

	if apiResp.Error != nil {
		return "", usage, obj.ErrAiErrorf("API error: %s", apiResp.Error.Message)
	}

	// Extract text from response
	for _, output := range apiResp.Output {
		if output.Type == "message" {
			for _, content := range output.Content {
				if content.Type == "output_text" || content.Type == "text" {
					return content.Text, usage, nil
				}
			}
		}
	}

	return "", usage, obj.ErrAiError("no text content in response")
}
