// package: db / database access and repository layer
// type:    data
// job:     create, update, and delete of games; play/clone counters.
// limits:  does not list or read games (-> game.go) or handle sessions (-> game_sessions.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/events"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
	"gopkg.in/yaml.v3"
)

// DeleteGame soft-deletes a game (sets deleted_at). userID must be the owner.
// Sessions referencing this game are preserved; they just won't show the game in listings.
func DeleteGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	// Load game and check permission
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return obj.ErrNotFound("game not found")
	}
	if err := canAccessGame(ctx, userID, OpDelete, game, nil); err != nil {
		return err
	}

	// Store workshop ID before deletion for event publishing
	workshopID := game.WorkshopID

	// Clean up game data: sessions, messages, tags, favourites, shares, game_shares + guest data
	_ = queries().DeleteGameSessionMessagesByGameID(ctx, gameID)
	_ = queries().DeleteGameSessionsByGameID(ctx, gameID)
	_ = queries().DeleteGameTagsByGameID(ctx, gameID)
	_ = queries().DeleteFavouritesByGameID(ctx, gameID)
	// Clean up game_share guest data before deleting game_shares
	_ = DeleteGuestDataByGameID(ctx, gameID)
	_ = queries().DeleteGameSharesByGameID(ctx, gameID)
	_ = queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true})

	if err := queries().SoftDeleteGame(ctx, gameID); err != nil {
		return err
	}

	// Publish game_deleted event if game belonged to a workshop
	if workshopID != nil {
		events.GetBroker().PublishGameDeleted(*workshopID, gameID, userID)
	}

	return nil
}

// CreateGame creates a new game. userID is set as the owner (createdBy).
// If game.WorkshopID is set, validates that user has read access to that workshop.
// For participants, automatically associates the game with their workshop.
func CreateGame(ctx context.Context, userID uuid.UUID, game *obj.Game) error {
	// Check if user can create games (requires authentication)
	if err := canAccessGame(ctx, userID, OpCreate, nil, nil); err != nil {
		return err
	}

	// If no workshop specified, auto-assign user's workshop (for participants)
	// Track if we auto-assigned so we can skip permission check (user always has access to their own workshop)
	autoAssigned := false
	if game.WorkshopID == nil {
		user, err := GetUserByID(ctx, userID)
		if err == nil && user.Role != nil && user.Role.Workshop != nil {
			game.WorkshopID = &user.Role.Workshop.ID
			autoAssigned = true
		}
	}

	// If workshop is specified (not auto-assigned), validate user has read access to the workshop
	if game.WorkshopID != nil && !autoAssigned {
		// Get the workshop to find its institution (use raw query, permission check follows)
		ws, err := queries().GetWorkshopByID(ctx, *game.WorkshopID)
		if err != nil {
			return obj.ErrForbidden("workshop not found")
		}
		// User must be able to see/read the workshop (participant, staff, or head)
		if err := canAccessWorkshop(ctx, userID, OpRead, ws.InstitutionID, game.WorkshopID, uuid.Nil); err != nil {
			return obj.ErrForbidden("not authorized to create games in this workshop")
		}
	}

	now := time.Now()
	game.ID = uuid.New()

	// Serialize theme to JSON if present
	var themeJSON pqtype.NullRawMessage
	if game.Theme != nil {
		themeBytes, err := json.Marshal(game.Theme)
		if err != nil {
			return obj.ErrServerError("failed to serialize theme")
		}
		themeJSON = pqtype.NullRawMessage{RawMessage: themeBytes, Valid: true}
	}

	arg := db.CreateGameParams{
		ID:                           game.ID,
		CreatedBy:                    uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:                    now,
		ModifiedBy:                   uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:                   now,
		Name:                         game.Name,
		Description:                  game.Description,
		Icon:                         game.Icon,
		WorkshopID:                   uuidPtrToNullUUID(game.WorkshopID),
		Public:                       game.Public,
		PublicSponsoredApiKeyShareID: uuidPtrToNullUUID(game.PublicSponsoredApiKeyShareID),
		SystemMessageScenario:        game.SystemMessageScenario,
		SystemMessageGameStart:       game.SystemMessageGameStart,
		ImageStyle:                   game.ImageStyle,
		Css:                          game.CSS,
		StatusFields:                 game.StatusFields,
		Theme:                        themeJSON,
		FirstMessage:                 sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:                  sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:                   game.FirstImage,
	}

	// Note: Private share hash is not generated at creation
	// Users must explicitly share the game after creating and writing the story

	_, err := queries().CreateGame(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return obj.ErrDuplicateNamef("A game with the name %q already exists", game.Name)
		}
		return err
	}

	// Publish game_created event if game belongs to a workshop
	if game.WorkshopID != nil {
		events.GetBroker().PublishGameCreated(*game.WorkshopID, game.ID, userID)
	}

	return nil
}

// UpdateGame updates an existing game. userID must be the owner.
func UpdateGame(ctx context.Context, userID uuid.UUID, game *obj.Game) error {
	// Load game and check permission (get both parsed and raw)
	existingGame, existingGameRaw, err := loadGameByIDWithRaw(ctx, game.ID)
	if err != nil {
		return err
	}
	if err := canAccessGame(ctx, userID, OpUpdate, existingGame, nil); err != nil {
		return err
	}

	now := time.Now()

	// Only the game creator can set Public to true.
	// Head/staff may unset it (set to false) but never enable it on another user's game.
	isOwner := existingGame.Meta.CreatedBy.Valid && existingGame.Meta.CreatedBy.UUID == userID
	if game.Public && !existingGame.Public && !isOwner {
		return obj.ErrForbidden("only the game creator can make a game public")
	}

	// If game is being set to private, clear public sponsorship
	if !game.Public {
		game.PublicSponsoredApiKeyShareID = nil
	}

	// Serialize theme to JSON if present
	var themeJSON pqtype.NullRawMessage
	if game.Theme != nil {
		themeBytes, err := json.Marshal(game.Theme)
		if err != nil {
			return obj.ErrServerError("failed to serialize theme")
		}
		themeJSON = pqtype.NullRawMessage{RawMessage: themeBytes, Valid: true}
	}

	arg := db.UpdateGameParams{
		ID:                           game.ID,
		CreatedBy:                    existingGameRaw.CreatedBy,
		CreatedAt:                    existingGameRaw.CreatedAt,
		ModifiedBy:                   uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:                   now,
		Name:                         game.Name,
		Description:                  game.Description,
		Icon:                         game.Icon,
		Public:                       game.Public,
		PublicSponsoredApiKeyShareID: uuidPtrToNullUUID(game.PublicSponsoredApiKeyShareID),
		SystemMessageScenario:        game.SystemMessageScenario,
		SystemMessageGameStart:       game.SystemMessageGameStart,
		ImageStyle:                   game.ImageStyle,
		Css:                          game.CSS,
		StatusFields:                 game.StatusFields,
		Theme:                        themeJSON,
		FirstMessage:                 sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:                  sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:                   game.FirstImage,
	}

	_, err = queries().UpdateGame(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return obj.ErrDuplicateNamef("A game with the name %q already exists", game.Name)
		}
		return err
	}

	// Publish game_updated event if game belongs to a workshop
	if existingGame.WorkshopID != nil {
		events.GetBroker().PublishGameUpdated(*existingGame.WorkshopID, game.ID, userID)
	}

	return nil
}

// UpdateGameYaml updates a game from YAML content. userID must be the owner.
func UpdateGameYaml(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, yamlContent string) error {
	log.Debug("UpdateGameYaml: starting", "user_id", userID, "game_id", gameID)

	// Get existing game first (includes permission check)
	existing, err := GetGameByID(ctx, &userID, gameID)
	if err != nil {
		log.Debug("UpdateGameYaml: GetGameByID failed", "error", err)
		return fmt.Errorf("game not found: %w", err)
	}
	log.Debug("UpdateGameYaml: existing game loaded", "name", existing.Name)

	// Additional permission check for update operation
	if err := canAccessGame(ctx, userID, OpUpdate, existing, nil); err != nil {
		return err
	}

	// Parse YAML into a game object
	var incoming obj.Game
	if err := yaml.Unmarshal([]byte(yamlContent), &incoming); err != nil {
		log.Debug("UpdateGameYaml: YAML unmarshal failed", "error", err)
		return obj.ErrValidation("invalid YAML")
	}
	log.Debug("UpdateGameYaml: YAML parsed", "incoming_name", incoming.Name, "incoming_description", incoming.Description)

	// Selectively copy allowed fields
	existing.Name = incoming.Name
	existing.Description = incoming.Description
	existing.SystemMessageScenario = incoming.SystemMessageScenario
	existing.SystemMessageGameStart = incoming.SystemMessageGameStart
	existing.ImageStyle = incoming.ImageStyle

	// Normalize JSON fields
	existing.StatusFields = functional.NormalizeJson(incoming.StatusFields, &[]obj.StatusField{})
	existing.CSS = functional.NormalizeJson(incoming.CSS, &obj.CSS{})

	log.Debug("UpdateGameYaml: calling UpdateGame", "game_id", existing.ID, "name", existing.Name)
	if err := UpdateGame(ctx, userID, existing); err != nil {
		log.Debug("UpdateGameYaml: UpdateGame failed", "error", err)
		return err
	}
	log.Debug("UpdateGameYaml: success")
	return nil
}

// IncrementGameCloneCount increments the clone count for a game
func IncrementGameCloneCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGameCloneCount(ctx, gameID)
}

// Helper functions for converting between sql.Null* types and pointers

// IncrementGamePlayCount increments the play count of a game by 1.
func IncrementGamePlayCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGamePlayCount(ctx, gameID)
}
