// package: db / database access and repository layer
// type:    data
// job:     create, read, delete, and listing of game sessions.
// limits:  does not handle session messages (-> game_messages.go) or api-key resolution (-> game_session_apikey.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// CreateGameSession creates a new game session with minimal required parameters.
// The function loads game details and constructs the session object internally.
// Parameters:
// - userID: the user creating the session
// - game: the game being played
// - apiKeyID: the API key to use for AI calls
// - aiModel: the AI model to use
// - workshopID: optional workshop context
// - theme: optional visual theme for the game player UI
// - imageStyle: optional adapted image style (if empty, uses game.ImageStyle)
func CreateGameSession(ctx context.Context, userID uuid.UUID, game *obj.Game, apiKeyID uuid.UUID, aiModel string, workshopID *uuid.UUID, theme *obj.GameTheme, language string, imageStyle string) (*obj.GameSession, error) {
	// Validate workshop access and game permissions
	if err := canAccessGameSession(ctx, userID, OpCreate, nil, game.ID, workshopID); err != nil {
		return nil, err
	}

	// Load API key to get platform
	apiKey, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return nil, obj.ErrNotFound("api key not found")
	}

	// Serialize theme to JSON if present
	var themeJSON pqtype.NullRawMessage
	if theme != nil {
		themeBytes, err := json.Marshal(theme)
		if err != nil {
			return nil, obj.ErrServerError("failed to serialize theme")
		}
		themeJSON = pqtype.NullRawMessage{RawMessage: themeBytes, Valid: true}
	}

	// Use provided imageStyle if set, otherwise fall back to game.ImageStyle
	if imageStyle == "" {
		imageStyle = game.ImageStyle
	}

	now := time.Now()
	arg := db.CreateGameSessionParams{
		CreatedBy:    uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:    now,
		ModifiedBy:   uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:   now,
		GameID:       game.ID,
		UserID:       userID,
		WorkshopID:   uuidPtrToNullUUID(workshopID),
		ApiKeyID:     uuid.NullUUID{UUID: apiKeyID, Valid: true},
		AiPlatform:   apiKey.Platform,
		AiModel:      aiModel,
		AiSession:    []byte("{}"), // Empty JSON object as initial state
		ImageStyle:   imageStyle,
		Language:     language,
		StatusFields: game.StatusFields,
		Theme:        themeJSON,
	}

	result, err := queries().CreateGameSession(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to create session")
	}

	// Construct and return the session object
	return &obj.GameSession{
		ID: result.ID,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
		GameID:          result.GameID,
		GameName:        game.Name,
		GameDescription: game.Description,
		GameScenario:    game.SystemMessageScenario,
		UserID:          result.UserID,
		WorkshopID:      nullUUIDToPtr(result.WorkshopID),
		ApiKeyID:        nullUUIDToPtr(result.ApiKeyID),
		AiPlatform:      result.AiPlatform,
		AiModel:         result.AiModel,
		AiSession:       string(result.AiSession),
		ImageStyle:      result.ImageStyle,
		Language:        result.Language,
		StatusFields:    result.StatusFields,
		Theme:           theme,
	}, nil
}

// UpdateGameSessionOrganisationUnverified marks a session as having an unverified organisation
func UpdateGameSessionOrganisationUnverified(ctx context.Context, sessionID uuid.UUID, isUnverified bool) error {
	err := queries().UpdateGameSessionOrganisationUnverified(ctx, db.UpdateGameSessionOrganisationUnverifiedParams{
		ID:                       sessionID,
		IsOrganisationUnverified: isUnverified,
	})
	if err != nil {
		return obj.ErrServerError("failed to update session organisation status")
	}
	return nil
}

// GetGameSessionByIDForGuest returns a session by ID, validating only that it belongs to the given game.
// Used by guest play endpoints where access is proven by the share token, not by user identity.
func GetGameSessionByIDForGuest(ctx context.Context, sessionID uuid.UUID, expectedGameID uuid.UUID) (*obj.GameSession, error) {
	s, err := queries().GetGameSessionByID(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("session not found")
	}
	if s.GameID != expectedGameID {
		return nil, obj.ErrForbidden("session does not belong to this game")
	}

	session := &obj.GameSession{
		ID:           s.ID,
		GameID:       s.GameID,
		UserID:       s.UserID,
		ApiKeyID:     nullUUIDToPtr(s.ApiKeyID),
		AiPlatform:   s.AiPlatform,
		AiModel:      s.AiModel,
		AiSession:    string(s.AiSession),
		ImageStyle:   s.ImageStyle,
		Language:     s.Language,
		StatusFields: s.StatusFields,
		Meta: obj.Meta{
			CreatedBy:  s.CreatedBy,
			CreatedAt:  &s.CreatedAt,
			ModifiedBy: s.ModifiedBy,
			ModifiedAt: &s.ModifiedAt,
		},
	}

	if s.Theme.Valid && len(s.Theme.RawMessage) > 0 {
		var theme obj.GameTheme
		if err := json.Unmarshal(s.Theme.RawMessage, &theme); err == nil {
			if theme.Preset == "" {
				theme.Preset = "default"
			}
			session.Theme = &theme
		}
	}

	game, err := queries().GetGameByID(ctx, s.GameID)
	if err == nil {
		session.GameName = game.Name
		session.GameDescription = game.Description
		session.GameScenario = game.SystemMessageScenario
	}

	if s.ApiKeyID.Valid {
		key, err := queries().GetApiKeyByID(ctx, s.ApiKeyID.UUID)
		if err == nil {
			session.ApiKey = &obj.ApiKey{
				ID:               key.ID,
				UserID:           key.UserID,
				Name:             key.Name,
				Platform:         key.Platform,
				Key:              key.Key,
				IsDefault:        key.IsDefault,
				LastUsageSuccess: sqlNullBoolToMaybeBool(key.LastUsageSuccess),
				LastErrorCode:    sqlNullStringToMaybeString(key.LastErrorCode),
			}
		}
	}

	return session, nil
}

// GetGameSessionByID returns a single session by ID with its API key loaded
func GetGameSessionByID(ctx context.Context, userID *uuid.UUID, sessionID uuid.UUID) (*obj.GameSession, error) {
	s, err := queries().GetGameSessionByID(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("session not found")
	}

	// Sessions always require authentication
	if userID == nil {
		return nil, obj.ErrUnauthorized("authentication required to access sessions")
	}

	// Check permission
	sessionObj := &obj.GameSession{
		ID:         s.ID,
		UserID:     s.UserID,
		WorkshopID: nullUUIDToPtr(s.WorkshopID),
	}
	if err := canAccessGameSession(ctx, *userID, OpRead, sessionObj, s.GameID, sessionObj.WorkshopID); err != nil {
		return nil, err
	}

	session := &obj.GameSession{
		ID:           s.ID,
		GameID:       s.GameID,
		UserID:       s.UserID,
		ApiKeyID:     nullUUIDToPtr(s.ApiKeyID),
		AiPlatform:   s.AiPlatform,
		AiModel:      s.AiModel,
		AiSession:    string(s.AiSession),
		ImageStyle:   s.ImageStyle,
		Language:     s.Language,
		StatusFields: s.StatusFields,
		Meta: obj.Meta{
			CreatedBy:  s.CreatedBy,
			CreatedAt:  &s.CreatedAt,
			ModifiedBy: s.ModifiedBy,
			ModifiedAt: &s.ModifiedAt,
		},
	}

	// Parse theme from JSON if present
	if s.Theme.Valid && len(s.Theme.RawMessage) > 0 {
		var theme obj.GameTheme
		if err := json.Unmarshal(s.Theme.RawMessage, &theme); err == nil {
			// Default preset for old sessions that predate the preset-only model
			if theme.Preset == "" {
				theme.Preset = "default"
			}
			session.Theme = &theme
		}
	}

	// Load game info
	game, err := queries().GetGameByID(ctx, s.GameID)
	if err == nil {
		session.GameName = game.Name
		session.GameDescription = game.Description
		session.GameScenario = game.SystemMessageScenario
	}

	// Load API key (if present - may be null if the key was deleted)
	if s.ApiKeyID.Valid {
		key, err := queries().GetApiKeyByID(ctx, s.ApiKeyID.UUID)
		if err == nil {
			session.ApiKey = &obj.ApiKey{
				ID:               key.ID,
				UserID:           key.UserID,
				Name:             key.Name,
				Platform:         key.Platform,
				Key:              key.Key,
				IsDefault:        key.IsDefault,
				LastUsageSuccess: sqlNullBoolToMaybeBool(key.LastUsageSuccess),
				LastErrorCode:    sqlNullStringToMaybeString(key.LastErrorCode),
			}
		}
		// If key not found, leave ApiKey as nil - frontend will prompt for a new one
	}

	// Re-resolve prompt constraints live (per-Spielzug) via the single canonical resolver
	// (see ResolveConstraint). This authenticated session-load path carries no share context;
	// share-based play (guest or authenticated-via-share) re-resolves on its own token-gated
	// routes, which always have the share at hand.
	user, err := GetUserByID(ctx, s.UserID)
	if err == nil {
		c := ResolveConstraint(ctx, user, nil)
		session.PromptConstraints = c.Text
		session.PromptConstraintSource = c.Source
		session.PromptConstraintSourceName = c.SourceName
		session.PromptConstraintReasoning = c.Reasoning
	}

	return session, nil
}

// ClearGameSessionApiKey clears the API key reference from a session
// Used when an API key becomes invalid (billing not active, key revoked, etc.)
func ClearGameSessionApiKey(ctx context.Context, sessionID uuid.UUID) error {
	return queries().ClearGameSessionApiKeyByID(ctx, sessionID)
}

// UserSessionWithGame represents a session with its game name for display
type UserSessionWithGame struct {
	obj.GameSession
	GameName string `json:"gameName"`
}

// GetUserSessionsFilters contains filter options for user sessions
type GetUserSessionsFilters struct {
	Search    string // Search by game name
	SortField string // game, model, lastPlayed (default)
}

// sessionRowToUserSession converts a db row to UserSessionWithGame
func sessionRowToUserSession(id, gameID, userID uuid.UUID, apiKeyID uuid.NullUUID, aiPlatform, aiModel string, aiSession []byte, imageStyle string, createdBy, modifiedBy uuid.NullUUID, createdAt, modifiedAt time.Time, gameName string) UserSessionWithGame {
	return UserSessionWithGame{
		GameSession: obj.GameSession{
			ID:         id,
			GameID:     gameID,
			UserID:     userID,
			ApiKeyID:   nullUUIDToPtr(apiKeyID),
			AiPlatform: aiPlatform,
			AiModel:    aiModel,
			AiSession:  string(aiSession),
			ImageStyle: imageStyle,
			Meta: obj.Meta{
				CreatedBy:  createdBy,
				CreatedAt:  &createdAt,
				ModifiedBy: modifiedBy,
				ModifiedAt: &modifiedAt,
			},
		},
		GameName: gameName,
	}
}

// GetGameSessionsByUserID returns recent sessions for a user with game names
func GetGameSessionsByUserID(ctx context.Context, userID uuid.UUID, filters *GetUserSessionsFilters) ([]UserSessionWithGame, error) {
	search := ""
	sortField := "lastPlayed"
	if filters != nil {
		search = filters.Search
		if filters.SortField != "" {
			sortField = filters.SortField
		}
	}

	var sessions []UserSessionWithGame

	if search != "" {
		searchParam := sql.NullString{String: search, Valid: true}
		switch sortField {
		case "game":
			rows, err := queries().SearchGameSessionsByUserIDSortByGame(ctx, db.SearchGameSessionsByUserIDSortByGameParams{UserID: userID, Column2: searchParam})
			if err != nil {
				return nil, obj.ErrServerError("failed to get user sessions")
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		case "model":
			rows, err := queries().SearchGameSessionsByUserIDSortByModel(ctx, db.SearchGameSessionsByUserIDSortByModelParams{UserID: userID, Column2: searchParam})
			if err != nil {
				return nil, obj.ErrServerError("failed to get user sessions")
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		default:
			rows, err := queries().SearchGameSessionsByUserID(ctx, db.SearchGameSessionsByUserIDParams{UserID: userID, Column2: searchParam})
			if err != nil {
				return nil, obj.ErrServerError("failed to get user sessions")
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		}
	} else {
		switch sortField {
		case "game":
			rows, err := queries().GetGameSessionsByUserIDSortByGame(ctx, userID)
			if err != nil {
				return nil, obj.ErrServerError("failed to get user sessions")
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		case "model":
			rows, err := queries().GetGameSessionsByUserIDSortByModel(ctx, userID)
			if err != nil {
				return nil, obj.ErrServerError("failed to get user sessions")
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		default:
			rows, err := queries().GetGameSessionsByUserID(ctx, userID)
			if err != nil {
				return nil, obj.ErrServerError("failed to get user sessions")
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		}
	}

	return sessions, nil
}

// DeleteGameSession deletes a game session and all its messages. userID must be the owner.
func DeleteGameSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	// Check permission
	sessionObj, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpDelete, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	// Delete messages first (cascading)
	if err := queries().DeleteGameSessionMessagesBySessionID(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session messages: %w", err)
	}

	// Delete the session
	if err := queries().DeleteGameSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteUserGameSessions deletes all sessions for a user+game combination (used when restarting a game)
func DeleteUserGameSessions(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	// First get all sessions for this game to delete their messages
	sessions, err := queries().GetGameSessionsByGameID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	// Delete messages for sessions owned by this user
	for _, s := range sessions {
		if s.UserID == userID {
			if err := queries().DeleteGameSessionMessagesBySessionID(ctx, s.ID); err != nil {
				return fmt.Errorf("failed to delete session messages: %w", err)
			}
		}
	}

	// Delete the sessions
	return queries().DeleteUserGameSessions(ctx, db.DeleteUserGameSessionsParams{
		UserID: userID,
		GameID: gameID,
	})
}

// GetGameSessionsByGameID returns all sessions for a game (requires read access to game)
func GetGameSessionsByGameID(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) ([]obj.GameSession, error) {
	// Check if user has read access to the game
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGame(ctx, userID, OpRead, game, nil); err != nil {
		return nil, err
	}

	dbSessions, err := queries().GetGameSessionsByGameID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	sessions := make([]obj.GameSession, 0, len(dbSessions))
	for _, s := range dbSessions {
		sessions = append(sessions, obj.GameSession{
			ID:         s.ID,
			GameID:     s.GameID,
			UserID:     s.UserID,
			ApiKeyID:   nullUUIDToPtr(s.ApiKeyID),
			AiPlatform: s.AiPlatform,
			AiModel:    s.AiModel,
			AiSession:  string(s.AiSession),
			ImageStyle: s.ImageStyle,
			Meta: obj.Meta{
				CreatedBy:  s.CreatedBy,
				CreatedAt:  &s.CreatedAt,
				ModifiedBy: s.ModifiedBy,
				ModifiedAt: &s.ModifiedAt,
			},
		})
	}

	return sessions, nil
}

// loadSessionForPermissionCheck loads a session and returns a minimal obj.GameSession for permission checking
func loadSessionByID(ctx context.Context, sessionID uuid.UUID) (*obj.GameSession, error) {
	session, err := queries().GetGameSessionByID(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("session not found")
	}
	return &obj.GameSession{
		ID:         session.ID,
		UserID:     session.UserID,
		WorkshopID: nullUUIDToPtr(session.WorkshopID),
	}, nil
}
