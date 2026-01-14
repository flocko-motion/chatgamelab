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
			{ID: "mistral-large-latest", Name: "Mistral Large Latest", Description: "Most capable Mistral model"},
			{ID: "mistral-medium-latest", Name: "Mistral Medium Latest", Description: "Balanced performance"},
		},
	}
}

func (p *MistralPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage) error {
	// TODO: implement Mistral action execution
	return fmt.Errorf("Mistral ExecuteAction not implemented")
}

func (p *MistralPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	// TODO: implement Mistral story expansion
	return fmt.Errorf("Mistral ExpandStory not implemented")
}

func (p *MistralPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	// TODO: implement Mistral image generation
	return fmt.Errorf("Mistral GenerateImage not implemented")
}

// Translate translates the given JSON objects (stringified) to the target in a single API call
func (p *MistralPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, error) {
	originals := ""
	for i, original := range input {
		originals += fmt.Sprintf("Original #%d: \n%s\n\n", i+1, original)
	}

	// Create the request for translation
	requestBody := chatCompletionRequest{
		Model: "mistral-large-latest",
		Messages: []message{
			{
				Role:    "system",
				Content: "You are an expert in translation of json structured language files for games. Translate the given JSON object to the target language while preserving the exact same structure and keys. Only translate the string values. Return a valid JSON object. You get the original already in two languages, so that you have more context to understand the intention of each field.",
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
		return "", fmt.Errorf("failed to translate: %w", err)
	}

	// Parse the translated JSON
	var translated map[string]any
	if err := response.parseContent(&translated); err != nil {
		return "", fmt.Errorf("failed to parse translated JSON: %w", err)
	}

	return functional.MustAnyToJson(translated), nil
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

	models := make([]obj.AiModel, 0, len(response.Data))
	for _, model := range response.Data {
		// Skip non-chat models
		if !isChatModel(model.ID) {
			continue
		}

		models = append(models, obj.AiModel{
			ID:          model.ID,
			Name:        model.ID,
			Description: fmt.Sprintf("Mistral model: %s", model.ID),
		})
	}

	return models, nil
}

// isChatModel checks if a model supports chat completions
func isChatModel(modelID string) bool {
	// List of known non-chat model prefixes to skip
	nonChatPrefixes := []string{
		"embed-",
		"rerank-",
		"code-",
	}

	for _, prefix := range nonChatPrefixes {
		if len(modelID) > len(prefix) && modelID[:len(prefix)] == prefix {
			return false
		}
	}

	return true
}
