package openai

import (
	"bytes"
	"cgl/lang"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (p *OpenAiPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, msg *obj.GameSessionMessage) error {
	if session.ApiKey == nil {
		return fmt.Errorf("session has no API key")
	}

	// Parse the model session
	var modelSession ModelSession
	if err := json.Unmarshal([]byte(session.AiSession), &modelSession); err != nil {
		return fmt.Errorf("failed to parse model session: %w", err)
	}

	// Serialize the player action as JSON input
	actionInput, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Build the request
	req := ResponsesAPIRequest{
		Model: session.AiModel,
		Input: string(actionInput),
		Store: true,
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

	// Make the API call
	apiResponse, err := callResponsesAPI(ctx, session.ApiKey.Key, req)
	if err != nil {
		return fmt.Errorf("OpenAI API error: %w", err)
	}

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
	if err := json.Unmarshal([]byte(responseText), msg); err != nil {
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
	msg.GameSessionID = session.ID
	msg.Type = obj.GameSessionMessageTypeGame

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
