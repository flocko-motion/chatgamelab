package mistral

import (
	"cgl/apiclient"
	"cgl/functional"
	"cgl/game/stream"
	"cgl/lang"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type MistralPlatform struct{}

const (
	mistralBaseURL        = "https://api.mistral.ai/v1"
	mistralChatEndpoint   = "/chat/completions"
	mistralModelsEndpoint = "/models"
)

// chatResponse represents the response from Mistral's chat completion API
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// parseContent parses the JSON content from the first choice into the provided destination
func (r *chatResponse) parseContent(dest any) error {
	if len(r.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}
	return json.Unmarshal([]byte(r.Choices[0].Message.Content), dest)
}

// message represents a chat message in the request
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// responseFormat represents the response format structure
type responseFormat struct {
	Type string `json:"type"`
}

// chatCompletionRequest represents the request structure for chat completion
type chatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	Temperature    float64        `json:"temperature"`
	ResponseFormat responseFormat `json:"response_format"`
}

// newApi creates a new API client for Mistral with the given API key
func (p *MistralPlatform) newApi(apiKey string) *apiclient.Client {
	return apiclient.NewApi(mistralBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
}

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

func (p *MistralPlatform) ResolveModel(model string) string {
	models := p.GetPlatformInfo().Models
	for _, m := range models {
		if m.ID == model {
			return m.Model
		}
	}
	// fallback: medium tier
	return models[1].Model
}

func (p *MistralPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage) (obj.TokenUsage, error) {
	// TODO: implement Mistral action execution
	return obj.TokenUsage{}, fmt.Errorf("Mistral ExecuteAction not implemented")
}

func (p *MistralPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) (obj.TokenUsage, error) {
	// TODO: implement Mistral story expansion
	return obj.TokenUsage{}, fmt.Errorf("Mistral ExpandStory not implemented")
}

func (p *MistralPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	// TODO: implement Mistral image generation
	return fmt.Errorf("Mistral GenerateImage not implemented")
}

// Translate translates the given JSON objects (stringified) to the target in a single API call
func (p *MistralPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, obj.TokenUsage, error) {
	originals := ""
	for i, original := range input {
		originals += fmt.Sprintf("Original #%d: \n%s\n\n", i+1, original)
	}

	const translateModel = "mistral-small-latest"

	// Create the request for translation
	requestBody := chatCompletionRequest{
		Model: translateModel,
		Messages: []message{
			{
				Role:    "system",
				Content: lang.TranslateInstruction,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Translate this JSON to %s:\n\n%s", lang.GetLanguageName(targetLang), originals),
			},
		},
		Temperature: 0.3,
		ResponseFormat: responseFormat{
			Type: "json_object",
		},
	}

	// Make the API call
	client := p.newApi(apiKey)

	var response chatResponse

	if err := client.PostJson(ctx, mistralChatEndpoint, requestBody, &response); err != nil {
		return "", obj.TokenUsage{}, fmt.Errorf("failed to translate: %w", err)
	}

	// Parse the translated JSON
	var translated map[string]any
	if err := response.parseContent(&translated); err != nil {
		return "", obj.TokenUsage{}, fmt.Errorf("failed to parse translated JSON: %w", err)
	}

	return functional.MustAnyToJson(translated), obj.TokenUsage{}, nil
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

// endsWithFourDigits checks if a model ID ends with -XXXX pattern (4 digits)
func endsWithFourDigits(modelID string) bool {
	parts := strings.Split(modelID, "-")
	if len(parts) < 2 {
		return false
	}

	lastPart := parts[len(parts)-1]

	// Check if last part is exactly 4 digits
	if len(lastPart) == 4 {
		for _, ch := range lastPart {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true
	}

	return false
}

// isRelevantModel checks if a model supports chat completions
func isRelevantModel(modelID string) bool {
	// List of known non-chat model prefixes to skip
	nonChatPrefixes := []string{
		"embed-",
		"rerank-",
		"code-",
		"codestral-",
		"devstral-",
	}

	for _, prefix := range nonChatPrefixes {
		if len(modelID) > len(prefix) && modelID[:len(prefix)] == prefix {
			return false
		}
	}

	// Skip dated models (ending with -XXXX where X is a digit)
	if endsWithFourDigits(modelID) {
		return false
	}

	return true
}

// GenerateTheme generates a visual theme JSON for the game player UI
func (p *MistralPlatform) GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, obj.TokenUsage, error) {
	if session.ApiKey == nil {
		return "", obj.TokenUsage{}, fmt.Errorf("session has no API key")
	}

	client := p.newApi(session.ApiKey.Key)

	requestBody := chatCompletionRequest{
		Model: session.AiModel,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		ResponseFormat: responseFormat{
			Type: "json_object",
		},
	}

	var response chatResponse
	if err := client.PostJson(ctx, mistralChatEndpoint, requestBody, &response); err != nil {
		return "", obj.TokenUsage{}, fmt.Errorf("failed to generate theme: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", obj.TokenUsage{}, fmt.Errorf("no response from Mistral")
	}

	return response.Choices[0].Message.Content, obj.TokenUsage{}, nil
}
