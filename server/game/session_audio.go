// package: game / on-demand audio narration
// type:    logic
// job:     returns a scene's audio narration, generating it via TTS only when the player asks for it and it does not exist yet.
// limits:  does not resolve API keys or stream audio during a turn; the regular turn flow handles that (-> session_play.go).
package game

import (
	"context"
	"fmt"
	"net/http"

	"cgl/db"
	"cgl/game/ai"
	"cgl/game/stream"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// GenerateMessageAudio returns the audio narration of a game message. Audio that already exists is
// returned as is; otherwise it is generated via TTS and persisted, so every scene costs at most one
// TTS call. Narration is only generated for turns played with narration switched on, so this covers
// scenes played while muted (the latest one when the player switches narration on, older ones on click).
// The session's API key must already be resolved.
func GenerateMessageAudio(ctx context.Context, session *obj.GameSession, messageID uuid.UUID) ([]byte, *obj.HTTPError) {
	if session.ApiKey == nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeInvalidApiKey, "Session has no API key. Please select a new API key.")
	}

	// Serialize with the turn flow: a scene that is still being written becomes available
	// once its turn has finished, and two requests for the same scene generate audio only once.
	unlock := sessionLocks.Lock(session.ID)
	defer unlock()

	msg, err := db.GetGameSessionMessageByIDPublic(ctx, messageID)
	if err != nil || msg.GameSessionID != session.ID {
		return nil, obj.NewHTTPErrorWithCode(http.StatusNotFound, obj.ErrCodeNotFound, "Message not found")
	}
	if len(msg.Audio) > 0 {
		return msg.Audio, nil
	}
	if msg.Type != obj.GameSessionMessageTypeGame || !msg.HasAudioOut || msg.Message == "" {
		return nil, obj.NewHTTPErrorWithCode(http.StatusNotFound, obj.ErrCodeNotFound, "No narration available for this message")
	}

	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeInvalidPlatform, fmt.Sprintf("AI platform error: %v", err))
	}

	// No SSE consumer for this stream; the audio is returned directly and saved below.
	responseStream := &stream.Stream{
		MessageID: messageID,
		Chunks:    make(chan obj.GameSessionMessageChunk, 1),
	}
	audio, err := platform.GenerateAudio(ctx, session, msg.Message, responseStream)
	if err != nil {
		log.Warn("on-demand audio generation failed", "session_id", session.ID, "message_id", messageID, "error", err)
		return nil, obj.NewHTTPErrorWithCode(500, extractAIErrorCode(err), fmt.Sprintf("Audio generation failed: %v", err))
	}
	if len(audio) == 0 {
		return nil, obj.NewHTTPErrorWithCode(http.StatusNotFound, obj.ErrCodeNotFound, "No narration available for this message")
	}

	if err := db.UpdateGameSessionMessageAudio(ctx, session.UserID, messageID, audio); err != nil {
		log.Warn("failed to save on-demand audio", "session_id", session.ID, "message_id", messageID, "error", err)
	}
	log.Debug("on-demand audio generated", "session_id", session.ID, "message_id", messageID, "audio_bytes", len(audio))
	return audio, nil
}
