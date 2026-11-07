// package: game / the AI calls the v1 session code makes
// type:    logic
// job:     names what the v1 session code asks of an AI platform; no platform answers, so v1 sessions cannot run.
// limits:  goes with the v1 session code; the engine runs sessions.
package game

import (
	"context"

	"cgl/game/stream"
	"cgl/obj"
)

// aiPlatform is what the v1 session code calls on an AI platform.
type aiPlatform interface {
	ResolveModelInfo(tierID string) *modelInfo
	ExecuteAction(ctx context.Context, session *obj.GameSession, action obj.GameSessionMessage, response *obj.GameSessionMessage, gameSchema map[string]interface{}) (obj.TokenUsage, error)
	ExpandStory(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) (obj.TokenUsage, error)
	GenerateImage(ctx context.Context, session *obj.GameSession, response *obj.GameSessionMessage, responseStream *stream.Stream) error
	GenerateAudio(ctx context.Context, session *obj.GameSession, text string, responseStream *stream.Stream) ([]byte, error)
	Translate(ctx context.Context, apiKey string, input []string, targetLang string) (string, obj.TokenUsage, error)
	GenerateTheme(ctx context.Context, session *obj.GameSession, systemPrompt, userPrompt string) (string, obj.TokenUsage, error)
	TranscribeAudio(ctx context.Context, apiKey string, audioData []byte, mimeType string) (string, error)
	ToolQuery(ctx context.Context, apiKey string, prompt string) (string, error)
}

// modelInfo is what a v1 tier could do.
type modelInfo struct {
	SupportsImage    bool
	SupportsAudioIn  bool
	SupportsAudioOut bool
}

// getAiPlatform finds no platform: the v1 session code no longer runs.
func getAiPlatform(platformName string) (aiPlatform, error) {
	return nil, obj.ErrInvalidPlatformf("v1 sessions no longer run on %q", platformName)
}
