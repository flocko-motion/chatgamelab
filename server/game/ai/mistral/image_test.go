//go:build ai_tests

package mistral

import (
	"cgl/game/stream"
	"cgl/obj"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func getTestApiKey() string {
	path := filepath.Join(os.Getenv("HOME"), ".ai", "mistral", "api-keys", "current")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// TestGenerateImage creates a real image via the Mistral Conversations API
// with the image_generation tool and downloads it via the Files API.
// Requires a valid Mistral API key at ~/.ai/mistral/api-keys/current.
func TestGenerateImage(t *testing.T) {
	apiKey := getTestApiKey()
	if apiKey == "" {
		t.Skip("no Mistral API key available")
	}

	platform := &MistralPlatform{}

	// Set up a minimal session and response with an image prompt
	prompt := "a medieval castle on a hilltop at sunset"
	messageID := uuid.New()
	session := &obj.GameSession{
		ID:         uuid.New(),
		ApiKey:     &obj.ApiKey{Key: apiKey},
		AiModel:    obj.AiModelBalanced,
		ImageStyle: "simple illustration, minimalist",
	}
	response := &obj.GameSessionMessage{
		ID:          messageID,
		ImagePrompt: &prompt,
	}

	// Create a stream with a buffered channel to capture the image
	responseStream := &stream.Stream{
		MessageID: messageID,
		Chunks:    make(chan obj.GameSessionMessageChunk, 100),
	}

	err := platform.GenerateImage(context.Background(), session, response, responseStream)
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}

	// Verify we got image data
	if len(response.Image) == 0 {
		t.Fatal("expected non-empty image data")
	}

	t.Logf("generated image: %d bytes", len(response.Image))

	// Optionally save to /tmp for manual inspection
	outPath := "/tmp/mistral-test-image.png"
	if err := os.WriteFile(outPath, response.Image, 0644); err != nil {
		t.Logf("could not save test image: %v", err)
	} else {
		t.Logf("saved test image to %s", outPath)
	}

	// Verify the stream received an image chunk
	select {
	case chunk := <-responseStream.Chunks:
		if !chunk.ImageDone {
			t.Errorf("expected ImageDone chunk, got: %+v", chunk)
		}
	default:
		t.Error("expected at least one chunk in the stream")
	}
}
