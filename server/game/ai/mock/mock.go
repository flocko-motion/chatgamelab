package mock

import (
	"bytes"
	"cgl/functional"
	"cgl/game/stream"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"time"

	lorem "github.com/drhodes/golorem"
)

type MockPlatform struct{}

func (p *MockPlatform) GetPlatformInfo() obj.AiPlatform {
	return obj.AiPlatform{
		ID:   "mock",
		Name: "Mock (Testing)",
		Models: []obj.AiModel{
			{ID: "mock-v1", Name: "Mock Model", Description: "Dummy AI for testing purposes only"},
		},
	}
}

func (p *MockPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage) error {
	// Parse status fields from session to generate mock status
	var statusFields []obj.StatusField
	if session != nil && session.StatusFields != "" {
		if err := json.Unmarshal([]byte(session.StatusFields), &statusFields); err != nil {
			return fmt.Errorf("failed to parse status fields: %w", err)
		}
	}

	// Generate mock values for each field
	mockStatus := make([]obj.StatusField, len(statusFields))
	for i, field := range statusFields {
		mockStatus[i] = obj.StatusField{
			Name:  field.Name,
			Value: fmt.Sprintf("%d", rand.Intn(100)),
		}
	}

	// Fill in the pre-created message with lorem ipsum text
	response.Message = lorem.Paragraph(3, 5)
	response.StatusFields = mockStatus
	response.ImagePrompt = functional.Ptr(lorem.Sentence(5, 10))
	response.GameSessionID = session.ID
	response.Type = obj.GameSessionMessageTypeGame

	return nil
}

// ExpandStory simulates streaming text expansion with mock lorem ipsum
func (p *MockPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	// Generate lorem ipsum text and stream it word by word
	fullText := lorem.Paragraph(5, 8)
	words := splitIntoChunks(fullText, 3) // Stream 3 words at a time

	for i, chunk := range words {
		isLast := i == len(words)-1
		responseStream.SendText(chunk+" ", isLast)
		time.Sleep(50 * time.Millisecond) // Simulate streaming delay
	}

	response.Message = fullText
	return nil
}

// GenerateImage simulates streaming image generation with mock images
func (p *MockPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
	if response.ImagePrompt == nil || *response.ImagePrompt == "" {
		return nil
	}

	// Send a low-res partial image first (simulates progressive refinement)
	partialImg := generateMockImage(8, 8)
	responseStream.SendImage(partialImg, false)
	time.Sleep(200 * time.Millisecond)

	// Send final high-res image
	finalImg := generateMockImage(32, 32)
	responseStream.SendImage(finalImg, true)
	response.Image = finalImg

	return nil
}

// Translate provides a mock translation implementation
func (p *MockPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, error) {
	// Mock implementation: generate pseudo-translations by adding "[MOCK]" prefix
	fmt.Printf("MOCK: Translating %d files to %s\n", len(input), targetLang)
	for _, file := range input {
		fmt.Printf("MOCK: Would process file: %s\n", file)
	}

	// Return mock translated JSON
	mockTranslation := `{"mock": "translation", "language": "` + targetLang + `"}`
	return mockTranslation, nil
}

// splitIntoChunks splits text into chunks of n words
func splitIntoChunks(text string, wordsPerChunk int) []string {
	words := make([]string, 0)
	current := ""
	count := 0
	for _, r := range text {
		if r == ' ' {
			count++
			if count >= wordsPerChunk {
				words = append(words, current)
				current = ""
				count = 0
				continue
			}
		}
		current += string(r)
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// generateMockImageWithSize creates a random PNG image of specified size
func generateMockImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rand.Intn(256)),
				G: uint8(rand.Intn(256)),
				B: uint8(rand.Intn(256)),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// ListModels returns mock models for testing
func (p *MockPlatform) ListModels(ctx context.Context, apiKey string) ([]obj.AiModel, error) {
	return []obj.AiModel{
		{ID: "mock-v1", Name: "Mock Model", Description: "Dummy AI for testing purposes"},
		{ID: "mock-v2", Name: "Mock Model v2", Description: "Enhanced dummy AI for testing"},
	}, nil
}
