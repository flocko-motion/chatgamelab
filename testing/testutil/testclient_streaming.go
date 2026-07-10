// package: testutil / SSE streaming helpers
// type:    logic
// job:     sends streaming session actions and consumes SSE message streams into StreamResult
// limits:  streaming/message transport only; no session creation (-> testclient_games)
package testutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"cgl/api/routes"
	"cgl/config"
	"cgl/obj"
)

// StreamResult holds the accumulated data from consuming an SSE message stream
type StreamResult struct {
	Text      string
	ImageData []byte
	AudioData []byte
}

// sendActionWithStream sends an action and consumes the SSE stream to get the full expanded story
func (u *UserClient) sendActionWithStream(sessionID string, req routes.SessionActionRequest) (obj.GameSessionMessage, *StreamResult, error) {
	u.t.Helper()

	// Send action
	var response obj.GameSessionMessage
	err := u.Post("sessions/"+sessionID, req, &response)
	if err != nil {
		return obj.GameSessionMessage{}, nil, err
	}

	// If not streaming, return the initial response
	if !response.Stream {
		return response, &StreamResult{Text: response.Message, ImageData: response.Image}, nil
	}

	// Consume the SSE stream to get full expanded story
	result, err := u.consumeMessageStream(response.ID.String())
	if err != nil {
		return obj.GameSessionMessage{}, nil, fmt.Errorf("failed to consume stream: %w", err)
	}

	// Update response with full content
	response.Message = result.Text
	response.Image = result.ImageData
	response.Stream = false

	return response, result, nil
}

// SendSystemMessage sends a system-type message to a game session (e.g., "init" for opening scene)
func (u *UserClient) SendSystemMessage(sessionID string, message string) (obj.GameSessionMessage, *StreamResult, error) {
	u.t.Helper()
	return u.sendActionWithStream(sessionID, routes.SessionActionRequest{
		Type:    "system",
		Message: message,
	})
}

// SendGameMessageWithStream sends a message and consumes the SSE stream to get the full expanded story
func (u *UserClient) SendGameMessageWithStream(sessionID string, message string) (obj.GameSessionMessage, *StreamResult, error) {
	u.t.Helper()
	return u.sendActionWithStream(sessionID, routes.SessionActionRequest{
		Message: message,
	})
}

// consumeMessageStream connects to SSE endpoint and consumes all chunks
func (u *UserClient) consumeMessageStream(messageID string) (*StreamResult, error) {
	u.t.Helper()

	serverURL, err := config.GetServerURL()
	if err != nil {
		return nil, fmt.Errorf("no server configured: %w", err)
	}

	url := fmt.Sprintf("%s/api/messages/%s/stream", serverURL, messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+u.Token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stream request failed with status %d", resp.StatusCode)
	}

	return parseSSEStream(resp.Body)
}

// parseSSEStream parses an SSE stream body and accumulates text, image, and audio data
func parseSSEStream(body io.Reader) (*StreamResult, error) {
	var fullText strings.Builder
	var imageData []byte
	var audioData []byte
	textDone := false
	imageDone := false
	audioDone := false
	scanner := bufio.NewScanner(body)
	// Increase buffer for large SSE lines (base64-encoded image data)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")
		var chunk obj.GameSessionMessageChunk
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse chunk: %w", err)
		}

		// Accumulate text
		if chunk.Text != "" {
			fullText.WriteString(chunk.Text)
		}

		// Accumulate image data
		if len(chunk.ImageData) > 0 {
			imageData = append(imageData, chunk.ImageData...)
		}

		// Accumulate audio data
		if len(chunk.AudioData) > 0 {
			audioData = append(audioData, chunk.AudioData...)
		}

		// Track completion
		if chunk.TextDone {
			textDone = true
		}
		if chunk.ImageDone {
			imageDone = true
		}
		if chunk.AudioDone {
			audioDone = true
		}

		// Check for error
		if chunk.Error != "" {
			return nil, fmt.Errorf("stream error: %s", chunk.Error)
		}

		// Stream is complete when all active channels are done
		if textDone && imageDone && audioDone {
			break
		}
		// Backwards-compatible: if no audio expected, text+image is enough
		if textDone && imageDone && len(audioData) == 0 && !audioDone {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return &StreamResult{
		Text:      fullText.String(),
		ImageData: imageData,
		AudioData: audioData,
	}, nil
}

// GetMessageAudio fetches the audio bytes for a message (composable high-level API)
func (u *UserClient) GetMessageAudio(messageID string) ([]byte, error) {
	u.t.Helper()

	serverURL, err := config.GetServerURL()
	if err != nil {
		return nil, fmt.Errorf("no server configured: %w", err)
	}

	url := fmt.Sprintf("%s/api/messages/%s/audio", serverURL, messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+u.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// consumeMessageStreamNoAuth connects to SSE endpoint and consumes all chunks (no auth required)
func consumeMessageStreamNoAuth(t *testing.T, messageID string) (*StreamResult, error) {
	t.Helper()

	serverURL, err := config.GetServerURL()
	if err != nil {
		return nil, fmt.Errorf("no server configured: %w", err)
	}

	url := fmt.Sprintf("%s/api/messages/%s/stream", serverURL, messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stream request failed with status %d", resp.StatusCode)
	}

	return parseSSEStream(resp.Body)
}
