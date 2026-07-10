// package: db / database access and repository layer
// type:    data
// job:     create, read, update, and delete of game session messages.
// limits:  does not manage sessions (-> game_sessions.go) or shares (-> game_shares.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/obj"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// CreateGameSessionMessage adds a message to a game session with auto-incremented seq
// Creating a message modifies the session, so we check OpUpdate permission
func CreateGameSessionMessage(ctx context.Context, userID uuid.UUID, msg obj.GameSessionMessage) (*obj.GameSessionMessage, error) {
	// Verify session access (creating messages = updating session)
	sessionObj, err := loadSessionByID(ctx, msg.GameSessionID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return nil, err
	}

	now := time.Now()
	var statusJSON sql.NullString
	if len(msg.StatusFields) > 0 {
		statusBytes, _ := json.Marshal(msg.StatusFields)
		statusJSON = sql.NullString{String: string(statusBytes), Valid: true}
	}

	arg := db.CreateGameSessionMessageParams{
		CreatedBy:                  uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:                  now,
		ModifiedBy:                 uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:                 now,
		GameSessionID:              msg.GameSessionID,
		Type:                       msg.Type,
		Message:                    msg.Message,
		Status:                     statusJSON,
		Plot:                       sql.NullString{String: functional.Deref(msg.Plot, ""), Valid: msg.Plot != nil},
		ImagePrompt:                sql.NullString{String: functional.Deref(msg.ImagePrompt, ""), Valid: msg.ImagePrompt != nil},
		Image:                      msg.Image,
		HasImage:                   msg.HasImage,
		HasAudio:                   msg.HasAudioOut,
		ApiKeyType:                 sql.NullString{String: msg.ApiKeyType, Valid: msg.ApiKeyType != ""},
		PromptConstraintSource:     sql.NullString{String: msg.PromptConstraintSource, Valid: msg.PromptConstraintSource != ""},
		PromptConstraintText:       sql.NullString{String: msg.PromptConstraintText, Valid: msg.PromptConstraintText != ""},
		PromptConstraintSourceName: sql.NullString{String: msg.PromptConstraintSourceName, Valid: msg.PromptConstraintSourceName != ""},
		PromptConstraintReasoning:  sql.NullString{String: msg.PromptConstraintReasoning, Valid: msg.PromptConstraintReasoning != ""},
	}

	result, err := queries().CreateGameSessionMessage(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to create session message")
	}

	// Return a copy with the generated values from the database
	msg.Seq = int(result.Seq)
	msg.ID = result.ID
	msg.Meta.CreatedAt = &result.CreatedAt
	msg.Meta.ModifiedAt = &result.ModifiedAt

	return &msg, nil
}

// CreateStreamingMessage creates a placeholder message with Stream=true for async AI responses
func CreateStreamingMessage(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, msgType string) (*obj.GameSessionMessage, error) {
	return CreateGameSessionMessage(ctx, userID, obj.GameSessionMessage{
		GameSessionID: sessionID,
		Type:          msgType,
		Stream:        true,
	})
}

// UpdateGameSessionMessage updates a message in the database
func UpdateGameSessionMessage(ctx context.Context, userID uuid.UUID, msg obj.GameSessionMessage) error {
	// Verify session ownership
	sessionObj, err := loadSessionByID(ctx, msg.GameSessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	now := time.Now()
	var statusJSON sql.NullString
	if len(msg.StatusFields) > 0 {
		statusBytes, _ := json.Marshal(msg.StatusFields)
		statusJSON = sql.NullString{String: string(statusBytes), Valid: true}
	}

	// Marshal token usage to JSON for storage
	var tokenUsageJSON pqtype.NullRawMessage
	if msg.TokenUsage != nil {
		tokenBytes, _ := json.Marshal(msg.TokenUsage)
		tokenUsageJSON = pqtype.NullRawMessage{RawMessage: tokenBytes, Valid: true}
	}

	arg := db.UpdateGameSessionMessageParams{
		ID:                         msg.ID,
		CreatedBy:                  uuid.NullUUID{},
		CreatedAt:                  time.Time{},
		ModifiedBy:                 uuid.NullUUID{},
		ModifiedAt:                 now,
		GameSessionID:              msg.GameSessionID,
		Type:                       msg.Type,
		Message:                    msg.Message,
		Status:                     statusJSON,
		Plot:                       sql.NullString{String: functional.Deref(msg.Plot, ""), Valid: msg.Plot != nil},
		ImagePrompt:                sql.NullString{String: functional.Deref(msg.ImagePrompt, ""), Valid: msg.ImagePrompt != nil},
		Image:                      msg.Image,
		HasImage:                   msg.HasImage,
		HasAudio:                   msg.HasAudioOut,
		PromptStatusUpdate:         sql.NullString{String: functional.Deref(msg.PromptStatusUpdate, ""), Valid: msg.PromptStatusUpdate != nil},
		PromptResponseSchema:       sql.NullString{String: functional.Deref(msg.PromptResponseSchema, ""), Valid: msg.PromptResponseSchema != nil},
		PromptImageGeneration:      sql.NullString{String: functional.Deref(msg.PromptImageGeneration, ""), Valid: msg.PromptImageGeneration != nil},
		PromptExpandStory:          sql.NullString{String: functional.Deref(msg.PromptExpandStory, ""), Valid: msg.PromptExpandStory != nil},
		ResponseRaw:                sql.NullString{String: functional.Deref(msg.ResponseRaw, ""), Valid: msg.ResponseRaw != nil},
		TokenUsage:                 tokenUsageJSON,
		UrlAnalytics:               sql.NullString{String: functional.Deref(msg.URLAnalytics, ""), Valid: msg.URLAnalytics != nil},
		ApiKeyType:                 sql.NullString{String: msg.ApiKeyType, Valid: msg.ApiKeyType != ""},
		PromptConstraintSource:     sql.NullString{String: msg.PromptConstraintSource, Valid: msg.PromptConstraintSource != ""},
		PromptConstraintText:       sql.NullString{String: msg.PromptConstraintText, Valid: msg.PromptConstraintText != ""},
		PromptConstraintSourceName: sql.NullString{String: msg.PromptConstraintSourceName, Valid: msg.PromptConstraintSourceName != ""},
		PromptConstraintReasoning:  sql.NullString{String: msg.PromptConstraintReasoning, Valid: msg.PromptConstraintReasoning != ""},
	}

	_, err = queries().UpdateGameSessionMessage(ctx, arg)
	if err != nil {
		return obj.ErrServerError("failed to update session message")
	}

	return nil
}

// UpdateGameSessionMessageImage updates only the image field of a message
func UpdateGameSessionMessageImage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID, image []byte) error {
	// Get message to find session
	msg, err := queries().GetGameSessionMessageByID(ctx, messageID)
	if err != nil {
		return obj.ErrNotFound("message not found")
	}
	// Verify session ownership
	sessionObj, err := loadSessionByID(ctx, msg.GameSessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	_, err = queries().UpdateGameSessionMessageImage(ctx, db.UpdateGameSessionMessageImageParams{
		ID:    messageID,
		Image: image,
	})
	if err != nil {
		return obj.ErrServerError("failed to update message image")
	}
	return nil
}

// UpdateGameSessionMessageAudio updates only the audio field of a message
func UpdateGameSessionMessageAudio(ctx context.Context, userID uuid.UUID, messageID uuid.UUID, audio []byte) error {
	// Get message to find session
	msg, err := queries().GetGameSessionMessageByID(ctx, messageID)
	if err != nil {
		return obj.ErrNotFound("message not found")
	}
	// Verify session ownership
	sessionObj, err := loadSessionByID(ctx, msg.GameSessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	_, err = queries().UpdateGameSessionMessageAudio(ctx, db.UpdateGameSessionMessageAudioParams{
		ID:    messageID,
		Audio: audio,
	})
	if err != nil {
		return obj.ErrServerError("failed to update message audio")
	}
	return nil
}

// GetGameSessionMessageAudioByID returns just the audio data for a message (public, no auth)
func GetGameSessionMessageAudioByID(ctx context.Context, messageID uuid.UUID) ([]byte, error) {
	row, err := queries().GetGameSessionMessageAudioByID(ctx, messageID)
	if err != nil {
		return nil, obj.ErrNotFound("message not found")
	}
	return row.Audio, nil
}

// inferCapabilityFlags ensures HasImage/HasAudio are true when actual data exists.
// Handles old messages created before the has_image/has_audio columns were added.
func inferCapabilityFlags(msg *obj.GameSessionMessage) {
	if !msg.HasImage && len(msg.Image) > 0 {
		msg.HasImage = true
	}
	if !msg.HasAudioOut && len(msg.Audio) > 0 {
		msg.HasAudioOut = true
	}
}

// mapAiInsightFields copies AI insight fields from the sqlc model to the obj model.
func mapAiInsightFields(msg *obj.GameSessionMessage, m db.GameSessionMessage) {
	if m.PromptStatusUpdate.Valid {
		msg.PromptStatusUpdate = &m.PromptStatusUpdate.String
	}
	if m.PromptResponseSchema.Valid {
		msg.PromptResponseSchema = &m.PromptResponseSchema.String
	}
	if m.PromptImageGeneration.Valid {
		msg.PromptImageGeneration = &m.PromptImageGeneration.String
	}
	if m.PromptExpandStory.Valid {
		msg.PromptExpandStory = &m.PromptExpandStory.String
	}
	if m.ResponseRaw.Valid {
		msg.ResponseRaw = &m.ResponseRaw.String
	}
	if m.UrlAnalytics.Valid {
		msg.URLAnalytics = &m.UrlAnalytics.String
	}
	if m.TokenUsage.Valid {
		var tu obj.TokenUsage
		if err := json.Unmarshal(m.TokenUsage.RawMessage, &tu); err == nil {
			msg.TokenUsage = &tu
		}
	}
	if m.ApiKeyType.Valid {
		msg.ApiKeyType = m.ApiKeyType.String
	}
	if m.PromptConstraintSource.Valid {
		msg.PromptConstraintSource = m.PromptConstraintSource.String
	}
	if m.PromptConstraintText.Valid {
		msg.PromptConstraintText = m.PromptConstraintText.String
	}
	if m.PromptConstraintSourceName.Valid {
		msg.PromptConstraintSourceName = m.PromptConstraintSourceName.String
	}
	if m.PromptConstraintReasoning.Valid {
		msg.PromptConstraintReasoning = m.PromptConstraintReasoning.String
	}
}

// GetGameSessionMessageImageByID returns just the image for a message (no auth required)
// Used for <img> tags which cannot send Authorization headers
// Security relies on message UUIDs being random/unguessable
func GetGameSessionMessageImageByID(ctx context.Context, messageID uuid.UUID) (*obj.GameSessionMessage, error) {
	m, err := queries().GetGameSessionMessageByID(ctx, messageID)
	if err != nil {
		return nil, obj.ErrNotFound("message not found")
	}

	return &obj.GameSessionMessage{
		ID:    m.ID,
		Image: m.Image,
	}, nil
}

// GetGameSessionMessageByIDPublic returns message fields needed for the status endpoint (no auth required).
// Security relies on message UUIDs being random/unguessable, same as image endpoint.
func GetGameSessionMessageByIDPublic(ctx context.Context, messageID uuid.UUID) (*obj.GameSessionMessage, error) {
	m, err := queries().GetGameSessionMessageByID(ctx, messageID)
	if err != nil {
		return nil, obj.ErrNotFound("message not found")
	}

	msg := &obj.GameSessionMessage{
		ID:          m.ID,
		Type:        m.Type,
		Message:     m.Message,
		Image:       m.Image,
		Audio:       m.Audio,
		HasImage:    m.HasImage,
		HasAudioOut: m.HasAudio,
	}

	// Parse status fields from JSON
	if m.Status.Valid && m.Status.String != "" {
		_ = json.Unmarshal([]byte(m.Status.String), &msg.StatusFields)
	}

	// Set plot and image prompt
	if m.Plot.Valid {
		msg.Plot = &m.Plot.String
	}
	if m.ImagePrompt.Valid {
		msg.ImagePrompt = &m.ImagePrompt.String
	}

	mapAiInsightFields(msg, m)
	inferCapabilityFlags(msg)

	return msg, nil
}

// GetLatestGameSessionMessage returns the most recent message for a session (requires read access to session)
func GetLatestGameSessionMessage(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*obj.GameSessionMessage, error) {
	// Check if user has read access to the session
	sessionObj, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGameSession(ctx, userID, OpRead, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return nil, err
	}

	m, err := queries().GetLatestGameSessionMessage(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("latest message not found")
	}

	msg := &obj.GameSessionMessage{
		ID:            m.ID,
		GameSessionID: m.GameSessionID,
		Seq:           int(m.Seq),
		Type:          m.Type,
		Message:       m.Message,
		HasImage:      m.HasImage,
		HasAudioOut:   m.HasAudio,
		Meta: obj.Meta{
			CreatedBy:  m.CreatedBy,
			CreatedAt:  &m.CreatedAt,
			ModifiedBy: m.ModifiedBy,
			ModifiedAt: &m.ModifiedAt,
		},
	}

	// Parse status fields from JSON
	if m.Status.Valid && m.Status.String != "" {
		_ = json.Unmarshal([]byte(m.Status.String), &msg.StatusFields)
	}

	// Set plot and image prompt
	if m.Plot.Valid {
		msg.Plot = &m.Plot.String
	}
	if m.ImagePrompt.Valid {
		msg.ImagePrompt = &m.ImagePrompt.String
	}

	mapAiInsightFields(msg, m)
	inferCapabilityFlags(msg)

	return msg, nil
}

// GetAllGameSessionMessages returns all messages for a session ordered by sequence (requires read access to session)
func GetAllGameSessionMessages(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) ([]obj.GameSessionMessage, error) {
	// Check if user has read access to the session
	sessionObj, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGameSession(ctx, userID, OpRead, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return nil, err
	}

	messages, err := queries().GetAllGameSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrServerError("failed to get session messages")
	}

	result := make([]obj.GameSessionMessage, 0, len(messages))
	for _, m := range messages {
		msg := obj.GameSessionMessage{
			ID:            m.ID,
			GameSessionID: m.GameSessionID,
			Seq:           int(m.Seq),
			Type:          m.Type,
			Message:       m.Message,
			Image:         m.Image,
			Audio:         m.Audio,
			HasImage:      m.HasImage,
			HasAudioOut:   m.HasAudio,
			Meta: obj.Meta{
				CreatedBy:  m.CreatedBy,
				CreatedAt:  &m.CreatedAt,
				ModifiedBy: m.ModifiedBy,
				ModifiedAt: &m.ModifiedAt,
			},
		}

		// Parse status fields from JSON
		if m.Status.Valid && m.Status.String != "" {
			_ = json.Unmarshal([]byte(m.Status.String), &msg.StatusFields)
		}

		// Set plot and image prompt
		if m.Plot.Valid {
			msg.Plot = &m.Plot.String
		}
		if m.ImagePrompt.Valid {
			msg.ImagePrompt = &m.ImagePrompt.String
		}

		mapAiInsightFields(&msg, m)
		inferCapabilityFlags(&msg)

		result = append(result, msg)
	}

	return result, nil
}

// GetLatestGuestSessionMessage returns the latest message for a guest session (no user permission check).
// Access must be validated by the share token at the route level.
func GetLatestGuestSessionMessage(ctx context.Context, sessionID uuid.UUID) (*obj.GameSessionMessage, error) {
	m, err := queries().GetLatestGameSessionMessage(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("latest message not found")
	}

	msg := &obj.GameSessionMessage{
		ID:            m.ID,
		GameSessionID: m.GameSessionID,
		Seq:           int(m.Seq),
		Type:          m.Type,
		Message:       m.Message,
		HasImage:      m.HasImage,
		HasAudioOut:   m.HasAudio,
		Meta: obj.Meta{
			CreatedBy:  m.CreatedBy,
			CreatedAt:  &m.CreatedAt,
			ModifiedBy: m.ModifiedBy,
			ModifiedAt: &m.ModifiedAt,
		},
	}
	if m.Status.Valid && m.Status.String != "" {
		_ = json.Unmarshal([]byte(m.Status.String), &msg.StatusFields)
	}
	if m.Plot.Valid {
		msg.Plot = &m.Plot.String
	}
	if m.ImagePrompt.Valid {
		msg.ImagePrompt = &m.ImagePrompt.String
	}
	mapAiInsightFields(msg, m)
	inferCapabilityFlags(msg)
	return msg, nil
}

// GetAllGuestSessionMessages returns all messages for a guest session (no user permission check).
// Access must be validated by the share token at the route level.
func GetAllGuestSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]obj.GameSessionMessage, error) {
	messages, err := queries().GetAllGameSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrServerError("failed to get session messages")
	}

	result := make([]obj.GameSessionMessage, 0, len(messages))
	for _, m := range messages {
		msg := obj.GameSessionMessage{
			ID:            m.ID,
			GameSessionID: m.GameSessionID,
			Seq:           int(m.Seq),
			Type:          m.Type,
			Message:       m.Message,
			Image:         m.Image,
			Audio:         m.Audio,
			HasImage:      m.HasImage,
			HasAudioOut:   m.HasAudio,
			Meta: obj.Meta{
				CreatedBy:  m.CreatedBy,
				CreatedAt:  &m.CreatedAt,
				ModifiedBy: m.ModifiedBy,
				ModifiedAt: &m.ModifiedAt,
			},
		}
		if m.Status.Valid && m.Status.String != "" {
			_ = json.Unmarshal([]byte(m.Status.String), &msg.StatusFields)
		}
		if m.Plot.Valid {
			msg.Plot = &m.Plot.String
		}
		if m.ImagePrompt.Valid {
			msg.ImagePrompt = &m.ImagePrompt.String
		}
		mapAiInsightFields(&msg, m)
		inferCapabilityFlags(&msg)
		result = append(result, msg)
	}
	return result, nil
}

// DeleteGameSessionMessage deletes a single message by ID.
// Used to clean up placeholder messages when AI actions fail.
func DeleteGameSessionMessage(ctx context.Context, messageID uuid.UUID) error {
	return queries().DeleteGameSessionMessage(ctx, messageID)
}

// CountGameSessionMessages returns the number of messages in a game session.
func CountGameSessionMessages(ctx context.Context, sessionID uuid.UUID) (int, error) {
	count, err := queries().CountGameSessionMessages(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
