// package: db / database access and repository layer
// type:    data
// job:     create, update, and delete of games; play/clone counters.
// limits:  does not list or read games (-> game.go).
package db

import (
	"cgl/events"
	"cgl/functional"
	"cgl/obj"
	"context"
	"database/sql"
	"engine"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

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

	genre, err := gameGenre(game.Genre, string(engine.GenreAdventure))
	if err != nil {
		return err
	}
	game.Genre = genre

	now := time.Now()
	game.ID = uuid.New()

	arg := CreateGameParams{
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
		Theme:                        stringPtrToNullString(game.Theme),
		FirstMessage:                 sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:                  sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:                   game.FirstImage,
		Genre:                        game.Genre,
	}

	// Note: Private share hash is not generated at creation
	// Users must explicitly share the game after creating and writing the story

	_, err = queries().CreateGame(ctx, arg)
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

	// A game sent without a genre keeps its own.
	genre, err := gameGenre(game.Genre, existingGame.Genre)
	if err != nil {
		return err
	}
	game.Genre = genre

	now := time.Now()

	// Whoever may edit a workshop game may publish it; a game outside a workshop only its creator.
	isOwner := existingGame.Meta.CreatedBy.Valid && existingGame.Meta.CreatedBy.UUID == userID
	if game.Public && !existingGame.Public && !isOwner && existingGame.WorkshopID == nil {
		return obj.ErrForbidden("only the game creator can make a game public")
	}

	// If game is being set to private, clear public sponsorship
	if !game.Public {
		game.PublicSponsoredApiKeyShareID = nil
	}

	arg := UpdateGameParams{
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
		Theme:                        stringPtrToNullString(game.Theme),
		FirstMessage:                 sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:                  sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:                   game.FirstImage,
		Genre:                        game.Genre,
	}

	_, err = queries().UpdateGame(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return obj.ErrDuplicateNamef("A game with the name %q already exists", game.Name)
		}
		return err
	}

	if existingGame.Public && !game.Public {
		DeleteGamePublicPageShares(ctx, game.ID)
	}

	// Publish game_updated event if game belongs to a workshop
	if existingGame.WorkshopID != nil {
		events.GetBroker().PublishGameUpdated(*existingGame.WorkshopID, game.ID, userID)
	}

	return nil
}

// UpdateGameFromFile replaces a game's content with a game file's. userID
// must be allowed to update the game.
func UpdateGameFromFile(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, file obj.GameFile) error {
	// Fetching the game checks the caller may see it.
	existing, err := GetGameByID(ctx, &userID, gameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}
	if err := canAccessGame(ctx, userID, OpUpdate, existing, nil); err != nil {
		return err
	}

	existing.Name = file.Name
	existing.Description = file.Description
	existing.SystemMessageScenario = file.SystemMessageScenario
	existing.SystemMessageGameStart = file.SystemMessageGameStart
	existing.ImageStyle = file.ImageStyle
	// A game file without a genre is an adventure, as every game was before genres.
	existing.Genre = file.Genre
	if existing.Genre == "" {
		existing.Genre = string(engine.GenreAdventure)
	}
	existing.Theme = file.Theme
	existing.StatusFields = functional.NormalizeJson(file.StatusFields, &[]obj.StatusField{})
	existing.CSS = functional.NormalizeJson(file.CSS, &obj.CSS{})

	return UpdateGame(ctx, userID, existing)
}

// gameGenre checks a game's genre against the engine's, taking fallback for
// an empty one.
func gameGenre(genre, fallback string) (string, error) {
	if genre == "" {
		genre = fallback
	}
	if !slices.Contains(engine.Genres(), engine.Genre(genre)) {
		return "", obj.ErrValidationf("unknown genre %q", genre)
	}
	return genre, nil
}

// IncrementGameCloneCount increments the clone count for a game
func IncrementGameCloneCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGameCloneCount(ctx, gameID)
}
