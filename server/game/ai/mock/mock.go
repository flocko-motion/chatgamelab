package mock

import (
	"bytes"
	"cgl/functional"
	"cgl/game/status"
	"cgl/game/stream"
	"cgl/obj"
	"context"
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
			{ID: obj.AiModelPremium, Name: "Mock Premium", Model: "mock-v1", Description: "Premium"},
			{ID: obj.AiModelBalanced, Name: "Mock Balanced", Model: "mock-v1", Description: "Balanced"},
			{ID: obj.AiModelEconomy, Name: "Mock Economy", Model: "mock-v1", Description: "Economy"},
		},
	}
}

func (p *MockPlatform) ResolveModelInfo(tierID string) *obj.AiModel {
	for _, m := range p.GetPlatformInfo().Models {
		if m.ID == tierID {
			return &m
		}
	}
	return nil
}

func (p *MockPlatform) ResolveModel(model string) string {
	models := p.GetPlatformInfo().Models
	for _, m := range models {
		if m.ID == model {
			return m.Model
		}
	}
	// fallback: balanced tier
	return models[1].Model
}

func (p *MockPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage, gameSchema map[string]interface{}) (obj.TokenUsage, error) {
	// Get field names from session to generate mock status
	fieldNames := status.FieldNames(session.StatusFields)

	// Generate mock values as a flat map (same format the AI would return)
	mockStatusMap := make(map[string]string, len(fieldNames))
	for _, name := range fieldNames {
		mockStatusMap[name] = fmt.Sprintf("%d", rand.Intn(100))
	}

	// Fill in the pre-created message with lorem ipsum text
	response.Message = lorem.Paragraph(3, 5)
	response.StatusFields = status.MapToFields(mockStatusMap, fieldNames, nil)
	response.ImagePrompt = functional.Ptr(lorem.Sentence(5, 10))
	response.GameSessionID = session.ID
	response.Type = obj.GameSessionMessageTypeGame

	return obj.TokenUsage{}, nil
}

// ExpandStory simulates streaming text expansion with mock lorem ipsum
func (p *MockPlatform) ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) (obj.TokenUsage, error) {
	// Generate lorem ipsum text and stream it word by word
	fullText := lorem.Paragraph(5, 8)
	words := splitIntoChunks(fullText, 3) // Stream 3 words at a time

	for i, chunk := range words {
		isLast := i == len(words)-1
		responseStream.SendText(chunk+" ", isLast)
		time.Sleep(50 * time.Millisecond) // Simulate streaming delay
	}

	response.Message = fullText
	return obj.TokenUsage{}, nil
}

// GenerateImage simulates streaming image generation with mock images
// Note: imagePrompt check is done in game_logic.go before calling this function
func (p *MockPlatform) GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error {
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

func (p *MockPlatform) GenerateAudio(ctx context.Context, session *obj.GameSession, text string, responseStream *stream.Stream) ([]byte, error) {
	// Generate a minimal MP3 frame header for test validation
	// 0xFF 0xFB = MP3 sync word (MPEG1 Layer3), followed by minimal frame data
	mp3Frame := []byte{
		0xFF, 0xFB, 0x90, 0x00, // MP3 frame header (sync word + MPEG1 Layer3 128kbps 44100Hz stereo)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // padding
	}

	responseStream.SendAudio(mp3Frame, true)
	return mp3Frame, nil
}

// Translate provides a mock translation implementation
func (p *MockPlatform) Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, obj.TokenUsage, error) {
	// Mock implementation: generate pseudo-translations by adding "[MOCK]" prefix
	fmt.Printf("MOCK: Translating %d files to %s\n", len(input), targetLang)
	for _, file := range input {
		fmt.Printf("MOCK: Would process file: %s\n", file)
	}

	// Return mock translated JSON
	mockTranslation := `{"mock": "translation", "language": "` + targetLang + `"}`
	return mockTranslation, obj.TokenUsage{}, nil
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

// GenerateTheme returns a mock theme JSON for testing
func (p *MockPlatform) GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, obj.TokenUsage, error) {
	// Return a random mock theme
	themes := []string{
		`{"corners":{"style":"brackets","color":"cyan"},"background":{"animation":"scanlines","tint":"cool"},"player":{"color":"cyan","indicator":"dot","monochrome":true,"showChevron":true},"thinking":{"text":"Processing...","style":"dots"},"typography":{"messages":"mono"},"statusEmojis":{"Health":"❤️","Energy":"⚡"}}`,
		`{"corners":{"style":"flourish","color":"amber"},"background":{"animation":"particles","tint":"warm"},"player":{"color":"amber","indicator":"diamond","monochrome":false,"showChevron":false},"thinking":{"text":"The tale continues...","style":"typewriter"},"typography":{"messages":"serif"},"statusEmojis":{"Health":"❤️","Gold":"🪙"}}`,
		`{"corners":{"style":"none","color":"slate"},"background":{"animation":"fog","tint":"dark"},"player":{"color":"rose","indicator":"none","monochrome":true,"showChevron":false},"thinking":{"text":"Something stirs...","style":"pulse"},"typography":{"messages":"serif"},"statusEmojis":{"Fear":"😨","Sanity":"🧠"}}`,
	}
	return themes[rand.Intn(len(themes))], obj.TokenUsage{}, nil
}
