package mistral

import (
	"bufio"
	"bytes"
	"cgl/apiclient"
	"cgl/game/stream"
	"cgl/log"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// extractResponseText extracts the text content from a Conversations API response
func extractResponseText(apiResponse *ConversationsAPIResponse) string {
	for _, output := range apiResponse.Outputs {
		if output.Role == "assistant" && output.Content != "" {
			return output.Content
		}
	}
	return ""
}

func callConversationsAPI(ctx context.Context, apiKey string, req ConversationsAPIRequest) (*ConversationsAPIResponse, obj.TokenUsage, error) {
	client := apiclient.NewApi(mistralBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})

	var apiResp ConversationsAPIResponse
	if err := client.PostJson(ctx, conversationsEndpoint, req, &apiResp); err != nil {
		return nil, obj.TokenUsage{}, err
	}

	usage := apiResp.Usage.toTokenUsage()
	log.Debug("API token usage", "input_tokens", usage.InputTokens, "output_tokens", usage.OutputTokens, "total_tokens", usage.TotalTokens)
	return &apiResp, usage, nil
}

func callConversationsAppendAPI(ctx context.Context, apiKey string, conversationID string, req ConversationsAppendRequest) (*ConversationsAPIResponse, obj.TokenUsage, error) {
	client := apiclient.NewApi(mistralBaseURL, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})

	endpoint := conversationsEndpoint + "/" + conversationID
	var apiResp ConversationsAPIResponse
	if err := client.PostJson(ctx, endpoint, req, &apiResp); err != nil {
		return nil, obj.TokenUsage{}, err
	}

	usage := apiResp.Usage.toTokenUsage()
	log.Debug("API token usage", "input_tokens", usage.InputTokens, "output_tokens", usage.OutputTokens, "total_tokens", usage.TotalTokens)
	return &apiResp, usage, nil
}

// callStreamingConversationsAPI makes a streaming call to the Conversations API
// Note: Uses direct HTTP instead of apiclient because it requires SSE streaming
func callStreamingConversationsAPI(ctx context.Context, apiKey string, conversationID string, req ConversationsAppendRequest, responseStream *stream.Stream) (fullText string, newConversationID string, usage obj.TokenUsage, err error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", "", obj.TokenUsage{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := mistralBaseURL + conversationsEndpoint + "/" + conversationID
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
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
	// Mistral streaming uses chat-completion-style events:
	// - data: {"choices":[{"delta":{"content":"..."}}]} for text chunks
	// - data: [DONE] for completion
	// The Conversations API may also include conversation_id and usage in a final event.
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

		// Try to parse as a conversation event with outputs (non-streaming final response)
		var convResp ConversationsAPIResponse
		if json.Unmarshal([]byte(data), &convResp) == nil && convResp.ConversationID != "" {
			newConversationID = convResp.ConversationID
			usage = convResp.Usage.toTokenUsage()
			// If there's content in outputs, append it
			for _, output := range convResp.Outputs {
				if output.Content != "" {
					textBuilder.WriteString(output.Content)
					responseStream.SendText(output.Content, false)
				}
			}
			continue
		}

		// Try chat-completion-style delta events
		var chatChunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *apiUsage `json:"usage,omitempty"`
		}
		if json.Unmarshal([]byte(data), &chatChunk) == nil {
			if len(chatChunk.Choices) > 0 && chatChunk.Choices[0].Delta.Content != "" {
				textBuilder.WriteString(chatChunk.Choices[0].Delta.Content)
				responseStream.SendText(chatChunk.Choices[0].Delta.Content, false)
			}
			if chatChunk.Usage != nil {
				usage = chatChunk.Usage.toTokenUsage()
			}
		}
	}

	log.Debug("streaming API token usage", "input_tokens", usage.InputTokens, "output_tokens", usage.OutputTokens, "total_tokens", usage.TotalTokens)
	// Signal text streaming complete
	responseStream.SendText("", true)
	return textBuilder.String(), newConversationID, usage, nil
}
