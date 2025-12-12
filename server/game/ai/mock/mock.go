package mock

import (
	"bytes"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
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

func (p *MockPlatform) InitGameSession(session *obj.GameSession, systemMessage string) error {
	// Mock implementation - does nothing
	return nil
}

func (p *MockPlatform) ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage) (*obj.GameSessionMessage, error) {
	// Parse status fields from session to generate mock status
	var statusFields []obj.StatusField
	if session != nil && session.StatusFields != "" {
		if err := json.Unmarshal([]byte(session.StatusFields), &statusFields); err != nil {
			return nil, fmt.Errorf("failed to parse status fields: %w", err)
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

	return &obj.GameSessionMessage{
		Seq:          action.Seq + 1,
		StatusFields: mockStatus,
		Image:        generateMockImage(),
	}, nil
}

// generateMockImage creates a random PNG image and returns it as a data URL
func generateMockImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	const width = 16
	const height = 16
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
