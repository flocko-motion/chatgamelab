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
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// extractResponseText extracts the assistant's text from a Conversations API response.
// Regular responses have content as a JSON string; tool responses have a ContentChunk array.
func extractResponseText(apiResponse *ConversationsAPIResponse) string {
	for _, output := range apiResponse.Outputs {
		if output.Role != "assistant" || len(output.Content) == 0 {
			continue
		}
		text := extractText(output.Content)
		if text != "" {
			return text
		}
	}
	return ""
}

// extractFileID extracts the first tool_file file_id from an image generation response
func extractFileID(apiResponse *ConversationsAPIResponse) string {
	for _, output := range apiResponse.Outputs {
		if output.Role != "assistant" || len(output.Content) == 0 {
			continue
		}
		chunks := parseContentChunks(output.Content)
		for _, chunk := range chunks {
			if chunk.Type == "tool_file" && chunk.FileID != "" {
				return chunk.FileID
			}
		}
	}
	return ""
}

// extractText returns the text from a content field.
// Dispatches based on the JSON type: string → return directly, array → join text chunks.
func extractText(raw json.RawMessage) string {
	switch raw[0] {
	case '"':
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	case '[':
		var sb strings.Builder
		for _, chunk := range parseContentChunks(raw) {
			if chunk.Type == "text" {
				sb.WriteString(chunk.Text)
			}
		}
		return sb.String()
	}
	return ""
}

// parseContentChunks unmarshals a JSON array of ContentChunk objects.
func parseContentChunks(raw json.RawMessage) []ContentChunk {
	var chunks []ContentChunk
	_ = json.Unmarshal(raw, &chunks)
	return chunks
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
	return &apiResp, usage, nil
}

// callStreamingConversationsAPI makes a streaming call to the Conversations API
// Note: Uses direct HTTP instead of apiclient because it requires SSE streaming
func callStreamingConversationsAPI(ctx context.Context, apiKey string, conversationID string, req ConversationsAppendRequest, responseStream *stream.Stream) (fullText string, newConversationID string, usage obj.TokenUsage, err error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to marshal request", err)
	}

	endpoint := mistralBaseURL + conversationsEndpoint + "/" + conversationID
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", "", obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", obj.TokenUsage{}, obj.ErrAiErrorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	// Mistral Conversations API streaming uses typed events:
	// - message.output.delta: text chunk with "content" field
	// - conversation.response.done: final event with "usage" (and optionally "conversation_id")
	// - data: [DONE] signals end of stream
	var textBuilder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// Peek at the event type
		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message.output.delta":
			var delta sseOutputDelta
			if err := json.Unmarshal([]byte(data), &delta); err == nil && delta.Content != "" {
				textBuilder.WriteString(delta.Content)
				responseStream.SendText(delta.Content, false)
			}
		case "conversation.response.done":
			var done sseResponseDone
			if err := json.Unmarshal([]byte(data), &done); err == nil {
				usage = done.Usage.toTokenUsage()
				if done.ConversationID != "" {
					newConversationID = done.ConversationID
				}
			}
		default:
			// Ignore other event types (e.g. conversation.response.started)
		}
	}

	// Signal text streaming complete
	responseStream.SendText("", true)
	return textBuilder.String(), newConversationID, usage, nil
}

// callImageConversationAPI creates a new conversation with the image_generation tool.
// This is a separate conversation from the game conversation - image generation requires
// the tool to be declared at conversation creation time.
func callImageConversationAPI(ctx context.Context, apiKey string, req ImageConversationRequest) (*ConversationsAPIResponse, error) {
	client := newApiClient(apiKey)

	var apiResp ConversationsAPIResponse
	if err := client.PostJson(ctx, conversationsEndpoint, req, &apiResp); err != nil {
		return nil, obj.WrapError(obj.ErrCodeAiError, "image conversation API failed", err)
	}

	log.Debug("image conversation API completed", "conversation_id", apiResp.ConversationID)
	return &apiResp, nil
}

// callTranscriptionAPI sends audio data to Mistral's /audio/transcriptions endpoint.
// Uses multipart form upload with model and file fields, returns the transcribed text.
func callTranscriptionAPI(ctx context.Context, apiKey string, audioData []byte, mimeType string) (string, error) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", mistralBaseURL+transcriptionEndpoint, &buf)
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

// downloadFile downloads file content from the Mistral Files API.
// Used to retrieve generated images by their file_id.
func downloadFile(ctx context.Context, apiKey string, fileID string) ([]byte, error) {
	endpoint := mistralBaseURL + filesEndpoint + "/" + fileID + "/content"

	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, obj.WrapError(obj.ErrCodeAiError, "failed to create download request", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, obj.WrapError(obj.ErrCodeAiError, "file download request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, obj.ErrAiErrorf("file download returned status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, obj.WrapError(obj.ErrCodeAiError, "failed to read file content", err)
	}

	log.Debug("file downloaded", "file_id", fileID, "size_bytes", len(data))
	return data, nil
}
