package mistral

import (
	"cgl/apiclient"
	"cgl/functional"
	"cgl/obj"
	"encoding/json"
	"strings"
)

const (
	mistralBaseURL        = "https://api.mistral.ai/v1"
	conversationsEndpoint = "/conversations"
	mistralModelsEndpoint = "/models"
	filesEndpoint         = "/files"
	transcriptionEndpoint = "/audio/transcriptions"
	transcriptionModel    = "voxtral-mini-latest"
	translateModel        = "mistral-small-latest"
	toolQueryModel        = "mistral-small-latest"
)

// ModelSession stores the Mistral conversation ID for conversation continuity
type ModelSession struct {
	ConversationID string `json:"conversationId"`
}

// InputMessage represents a single message in the Conversations API input array.
// Role can be "system" (instructions/reminders) or "user" (player actions).
type InputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ConversationsAPIRequest is the request body for creating a new conversation
type ConversationsAPIRequest struct {
	Model          string          `json:"model"`
	Inputs         []InputMessage  `json:"inputs"`
	Instructions   string          `json:"instructions,omitempty"`
	Store          bool            `json:"store"`
	Stream         bool            `json:"stream,omitempty"`
	CompletionArgs *CompletionArgs `json:"completion_args,omitempty"`
}

// ConversationsAppendRequest is the request body for appending to an existing conversation
type ConversationsAppendRequest struct {
	Inputs         []InputMessage  `json:"inputs"`
	Store          bool            `json:"store"`
	Stream         bool            `json:"stream,omitempty"`
	CompletionArgs *CompletionArgs `json:"completion_args,omitempty"`
}

// CompletionArgs holds whitelisted arguments from the chat completion API
type CompletionArgs struct {
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
}

// ResponseFormat specifies the output format for the model
type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema wraps a JSON schema for structured output
type JSONSchema struct {
	Name   string      `json:"name,omitempty"`
	Schema interface{} `json:"schema,omitempty"`
	Strict bool        `json:"strict,omitempty"`
}

// ConversationsAPIResponse is the response from the Conversations API
type ConversationsAPIResponse struct {
	ConversationID string        `json:"conversation_id"`
	Object         string        `json:"object"`
	Outputs        []OutputEntry `json:"outputs"`
	Usage          apiUsage      `json:"usage"`
}

// OutputEntry represents a single output entry from the Conversations API.
// Content is a string for regular text responses, but an array of ContentChunk
// for responses that include tool outputs (e.g. image generation).
type OutputEntry struct {
	Content json.RawMessage `json:"content"`
	Role    string          `json:"role"`
	Object  string          `json:"object"`
	Type    string          `json:"type"`
}

// ContentChunk represents a single chunk in a rich content array.
// For text: {"type": "text", "text": "..."}
// For tool_file: {"type": "tool_file", "tool": "image_generation", "file_id": "...", "file_name": "...", "file_type": "png"}
type ContentChunk struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Tool     string `json:"tool,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	FileName string `json:"file_name,omitempty"`
	FileType string `json:"file_type,omitempty"`
}

// ImageConversationRequest is the request body for creating a new conversation with tools.
// Used for image generation which requires the image_generation tool.
type ImageConversationRequest struct {
	Model          string          `json:"model"`
	Inputs         []InputMessage  `json:"inputs"`
	Instructions   string          `json:"instructions,omitempty"`
	Store          bool            `json:"store"`
	Tools          []Tool          `json:"tools"`
	CompletionArgs *CompletionArgs `json:"completion_args,omitempty"`
}

// Tool represents a tool available to the model during a conversation
type Tool struct {
	Type string `json:"type"`
}

// apiUsage matches Mistral's usage format
type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (u apiUsage) toTokenUsage() obj.TokenUsage {
	return obj.TokenUsage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
}

// SSE event types for streaming conversations
// message.output.delta: streamed text chunk
type sseOutputDelta struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// conversation.response.done: final event with usage
type sseResponseDone struct {
	Type           string   `json:"type"`
	ConversationID string   `json:"conversation_id,omitempty"`
	Usage          apiUsage `json:"usage"`
}

// newApi creates a new API client for Mistral with the given API key
func (p *MistralPlatform) newApi(apiKey string) *apiclient.Client {
	return apiclient.NewApi(mistralBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
}

// newApiClient creates a standalone API client for Mistral with the given API key
func newApiClient(apiKey string) *apiclient.Client {
	return apiclient.NewApi(mistralBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
}

// isRelevantModel checks if a model supports conversations
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
		if strings.HasPrefix(modelID, prefix) {
			return false
		}
	}

	// Skip dated models (ending with -XXXX where X is a digit)
	if functional.EndsWithDigits(modelID, 4) {
		return false
	}

	return true
}
