package mistral

import (
	"cgl/functional"
	"cgl/game/status"
	"cgl/game/stream"
	"cgl/game/templates"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

type MistralPlatform struct{}

func (p *MistralPlatform) GetPlatformInfo() obj.AiPlatform {
	return obj.AiPlatform{
		ID:   "mistral",
		Name: "Mistral",
		Models: []obj.AiModel{
			{ID: obj.AiModelPremium, Name: "Mistral Large", Model: "mistral-large-latest", Description: "Premium"},
			{ID: obj.AiModelBalanced, Name: "Mistral Medium", Model: "mistral-medium-latest", Description: "Balanced"},
			{ID: obj.AiModelEconomy, Name: "Mistral Small", Model: "mistral-small-latest", Description: "Economy"},
		},
	}
}

func (p *MistralPlatform) ResolveModelInfo(tierID string) *obj.AiModel {
	info := p.GetPlatformInfo()
	return info.ResolveModelWithDowngrade(tierID)
}

func (p *MistralPlatform) ResolveModel(tierID string) string {
	if m := p.ResolveModelInfo(tierID); m != nil {
		return m.Model
	}
	// fallback: medium tier
	return p.GetPlatformInfo().Models[1].Model
}

func (p *MistralPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage, gameSchema map[string]interface{}) (obj.TokenUsage, error) {
	model := p.ResolveModel(session.AiModel)
	log.Debug("Mistral ExecuteAction starting", "session_id", session.ID, "action_type", action.Type, "model", model)

	if session.ApiKey == nil {
		return obj.TokenUsage{}, fmt.Errorf("session has no API key")
	}

	// Parse the model session
	log.Debug("[TRACE] Mistral ExecuteAction AiSession input", "ai_session", session.AiSession, "action_type", action.Type)
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		log.Warn("failed to parse model session", "error", err, "ai_session_raw", session.AiSession)
		return obj.TokenUsage{}, fmt.Errorf("failed to parse model session: %w", err)
	}
	log.Debug("[TRACE] Mistral parsed ModelSession", "conversation_id", modelSession.ConversationID)

	// Serialize the player action as JSON input (minimal AI-facing structure)
	actionInput := action.ToAiJSON()

	completionArgs := &CompletionArgs{
		MaxTokens: 5000,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "game_response",
				Schema: gameSchema,
				Strict: true,
			},
		},
	}

	var apiResponse *ConversationsAPIResponse
	var usage obj.TokenUsage
	var err error

	if action.Type == obj.GameSessionMessageTypeSystem {
		// First turn: create a new conversation with system instructions
		req := ConversationsAPIRequest{
			Model:          model,
			Instructions:   action.Message,
			Inputs:         []InputMessage{{Role: "user", Content: templates.PromptMessageStart}},
			Store:          true,
			CompletionArgs: completionArgs,
		}
		apiResponse, usage, err = callConversationsAPI(ctx, session.ApiKey.Key, req)
	} else if modelSession.ConversationID != "" {
		// Continuation: append to existing conversation with developer reminder
		// Mistral append only accepts "user" or "assistant" roles (not "system")
		req := ConversationsAppendRequest{
			Inputs: []InputMessage{
				{Role: "user", Content: templates.ReminderExecuteAction},
				{Role: "user", Content: actionInput},
			},
			Store:          true,
			CompletionArgs: completionArgs,
		}
		apiResponse, usage, err = callConversationsAppendAPI(ctx, session.ApiKey.Key, modelSession.ConversationID, req)
		// Set debug prompt showing full input sent to the AI
		response.PromptStatusUpdate = functional.Ptr(
			"[system] " + templates.ReminderExecuteAction + "\n\n[user] " + actionInput)
	} else {
		return obj.TokenUsage{}, fmt.Errorf("no conversation ID and not a system message")
	}

	if err != nil {
		log.Error("Mistral API call failed", "error", err, "session_id", session.ID, "model", model)
		return obj.TokenUsage{}, fmt.Errorf("Mistral API error: %w", err)
	}
	log.Debug("Mistral API call completed", "conversation_id", apiResponse.ConversationID)

	// Extract the text response
	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return usage, fmt.Errorf("no text response from Mistral")
	}

	response.ResponseRaw = &responseText

	// Parse the AI response and convert to internal format
	log.Debug("parsing AI response", "response_length", len(responseText), "response_text", responseText)
	if err := status.ParseGameResponse(responseText, session.StatusFields, action.StatusFields, response); err != nil {
		log.Error("failed to parse game response", "error", err, "response_text", responseText)
		return usage, err
	}

	// Update model session with new conversation ID
	log.Debug("[TRACE] Mistral updating AiSession", "old_conversation_id", modelSession.ConversationID, "new_conversation_id", apiResponse.ConversationID)
	modelSession.ConversationID = apiResponse.ConversationID
	sessionJSON, err := json.Marshal(modelSession)
	if err != nil {
		return usage, fmt.Errorf("failed to marshal model session: %w", err)
	}
	session.AiSession = string(sessionJSON)
	log.Debug("[TRACE] Mistral AiSession after ExecuteAction", "ai_session", session.AiSession)

	// Set fields that come from the session, not from the AI
	response.GameSessionID = session.ID
	response.Type = obj.GameSessionMessageTypeGame

	return usage, nil
}

// ExpandStory expands the plot outline to full narrative text using streaming
func (p *MistralPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) (obj.TokenUsage, error) {
	log.Debug("Mistral ExpandStory starting", "session_id", session.ID, "message_id", response.ID)

	if session.ApiKey == nil {
		return obj.TokenUsage{}, fmt.Errorf("session has no API key")
	}

	// Parse the model session to get conversation ID
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		return obj.TokenUsage{}, fmt.Errorf("failed to parse model session: %w", err)
	}

	// Append to existing conversation with streaming - plain text, no JSON schema
	// Mistral append only accepts "user" or "assistant" roles (not "system")
	// Must explicitly set response_format to text to override the inherited json_schema from ExecuteAction
	req := ConversationsAppendRequest{
		Inputs: []InputMessage{
			{Role: "user", Content: templates.PromptNarratePlotOutline},
		},
		Store:  true,
		Stream: true,
		CompletionArgs: &CompletionArgs{
			ResponseFormat: &ResponseFormat{Type: "text"},
		},
	}

	// Make streaming API call
	log.Debug("calling Mistral streaming API for story expansion")
	fullText, newConversationID, usage, err := callStreamingConversationsAPI(ctx, session.ApiKey.Key, modelSession.ConversationID, req, responseStream)
	if err != nil {
		// For story expansion, partial text is still usable - don't fail if we got some output
		if len(fullText) > 0 {
			log.Warn("Mistral streaming API incomplete, using partial text",
				"error", err,
				"session_id", session.ID,
				"text_length", len(fullText),
			)
		} else {
			log.Error("Mistral streaming API failed",
				"error", err,
				"session_id", session.ID,
			)
			return usage, fmt.Errorf("Mistral streaming API error: %w", err)
		}
	}
	log.Debug("story expansion completed", "text_length", len(fullText), "new_conversation_id", newConversationID)

	// Update response with full text
	response.Message = fullText

	// Update model session with new conversation ID
	if newConversationID != "" {
		modelSession.ConversationID = newConversationID
	}
	sessionJSON, err := json.Marshal(modelSession)
	if err != nil {
		return usage, fmt.Errorf("failed to marshal model session: %w", err)
	}
	session.AiSession = string(sessionJSON)

	return usage, nil
}

// GenerateImage is a no-op for Mistral.
// NOTE: Mistral supports image generation only as a built-in agent tool via the Conversations API
// (see https://docs.mistral.ai/agents/tools/built-in/image_generation), not as a standalone endpoint.
// This would require creating an agent with tools=[{"type":"image_generation"}], parsing tool_file
// chunks from the response, and downloading the image via the files API. Not implemented yet.
func (p *MistralPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	log.Debug("Mistral GenerateImage skipped - not supported", "session_id", session.ID)
	return nil
}

func (p *MistralPlatform) GenerateAudio(ctx context.Context, session *obj.GameSession, text string, responseStream *stream.Stream) ([]byte, error) {
	log.Debug("Mistral GenerateAudio skipped - not supported", "session_id", session.ID)
	return nil, nil
}

// Translate translates the given JSON objects (stringified) to the target in a single API call
func (p *MistralPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, obj.TokenUsage, error) {
	originals := ""
	for i, original := range input {
		originals += fmt.Sprintf("Original #%d: \n%s\n\n", i+1, original)
	}

	req := ConversationsAPIRequest{
		Model:        translateModel,
		Instructions: lang.TranslateInstruction,
		Inputs:       []InputMessage{{Role: "user", Content: fmt.Sprintf("Translate this JSON to %s:\n\n%s", lang.GetLanguageName(targetLang), originals)}},
		Store:        false,
		CompletionArgs: &CompletionArgs{
			ResponseFormat: &ResponseFormat{
				Type: "json_object",
			},
		},
	}

	apiResponse, usage, err := callConversationsAPI(ctx, apiKey, req)
	if err != nil {
		return "", obj.TokenUsage{}, fmt.Errorf("failed to translate: %w", err)
	}

	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return "", usage, fmt.Errorf("no text response from Mistral")
	}

	// Validate it's valid JSON
	var translated map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &translated); err != nil {
		return "", usage, fmt.Errorf("failed to parse translated JSON: %w", err)
	}

	return functional.MustAnyToJson(translated), usage, nil
}

func (p *MistralPlatform) ListModels(ctx context.Context, apiKey string) ([]obj.AiModel, error) {
	client := p.newApi(apiKey)

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := client.GetJson(ctx, mistralModelsEndpoint, &response); err != nil {
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
			if !isRelevantModel(model.ID) {
				continue
			}
		}

		models = append(models, obj.AiModel{
			ID:          model.ID,
			Name:        model.ID,
			Description: fmt.Sprintf("Mistral model: %s", model.ID),
		})
	}

	// Sort alphabetically by model ID
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	return models, nil
}

// ToolQuery sends a single text prompt and returns a text answer using a fast model.
func (p *MistralPlatform) ToolQuery(ctx context.Context, apiKey string, prompt string) (string, error) {
	req := ConversationsAPIRequest{
		Model:  toolQueryModel,
		Inputs: []InputMessage{{Role: "user", Content: prompt}},
		Store:  false,
	}

	apiResponse, _, err := callConversationsAPI(ctx, apiKey, req)
	if err != nil {
		return "", fmt.Errorf("ToolQuery failed: %w", err)
	}

	text := extractResponseText(apiResponse)
	if text == "" {
		return "", fmt.Errorf("ToolQuery: no text in response")
	}

	return text, nil
}

// GenerateTheme generates a visual theme JSON for the game player UI
func (p *MistralPlatform) GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, obj.TokenUsage, error) {
	log.Debug("Mistral GenerateTheme starting", "session_id", session.ID)

	if session.ApiKey == nil {
		return "", obj.TokenUsage{}, fmt.Errorf("session has no API key")
	}

	model := p.ResolveModel(session.AiModel)
	req := ConversationsAPIRequest{
		Model:        model,
		Instructions: systemPrompt,
		Inputs:       []InputMessage{{Role: "user", Content: userPrompt}},
		Store:        false,
	}

	apiResponse, usage, err := callConversationsAPI(ctx, session.ApiKey.Key, req)
	if err != nil {
		return "", obj.TokenUsage{}, fmt.Errorf("failed to generate theme: %w", err)
	}

	responseText := extractResponseText(apiResponse)
	if responseText == "" {
		return "", usage, fmt.Errorf("no text response from Mistral")
	}

	log.Debug("Mistral GenerateTheme completed", "response_length", len(responseText))
	return responseText, usage, nil
}
