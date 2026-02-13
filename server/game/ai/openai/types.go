package openai

import (
	"cgl/apiclient"
	"cgl/functional"
	"cgl/obj"
	"strings"
)

const (
	openaiBaseURL         = "https://api.openai.com/v1"
	responsesEndpoint     = "/responses"
	modelsEndpoint        = "/models"
	imageGenEndpoint      = "/images/generations"
	speechEndpoint        = "/audio/speech"
	transcriptionEndpoint = "/audio/transcriptions"
	transcriptionModel    = "gpt-4o-mini-transcribe"
	translateModel        = "gpt-5.1-codex"
	toolQueryModel        = "gpt-5.1-codex"
	ttsModel              = "gpt-4o-mini-tts"
	ttsVoice              = "cedar" // openai recommends marin or cedar
	ttsFormat             = "mp3"
)

// ModelSession stores the OpenAI response ID for conversation continuity
type ModelSession struct {
	ResponseID string `json:"responseId"`
}

// InputMessage represents a single message in the Responses API input array.
// Role can be "developer" (instructions/reminders) or "user" (player actions).
type InputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponsesAPIRequest is the request body for the Responses API
type ResponsesAPIRequest struct {
	Model              string         `json:"model"`
	Input              []InputMessage `json:"input"`
	Instructions       string         `json:"instructions,omitempty"`
	PreviousResponseID string         `json:"previous_response_id,omitempty"`
	Store              bool           `json:"store"`
	Stream             bool           `json:"stream,omitempty"`
	MaxOutputTokens    int            `json:"max_output_tokens,omitempty"`
	Text               *TextConfig    `json:"text,omitempty"`
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

// apiTokenUsage matches OpenAI's snake_case JSON format
type apiTokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (u apiTokenUsage) toTokenUsage() obj.TokenUsage {
	return obj.TokenUsage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalTokens:  u.TotalTokens,
	}
}

// SSE event types for streaming responses
type sseEvent struct {
	Type string `json:"type"`
}

type sseTextDelta struct {
	Delta string `json:"delta"`
}

type sseResponseCompleted struct {
	Response struct {
		ID    string        `json:"id"`
		Usage apiTokenUsage `json:"usage"`
	} `json:"response"`
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
	Output []OutputItem  `json:"output"`
	Usage  apiTokenUsage `json:"usage"`
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
	return functional.EndsWithDigits(modelID, 4) || functional.EndsWithDatePattern(modelID)
}
