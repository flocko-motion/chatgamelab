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

type GetGamesFilters struct {
	PublicOnly bool
	Search     string
	SortField  string // name, createdAt, modifiedAt
	SortDir    string // asc, desc
	Filter     string // all, own, public
}

func userIsAllowedToPlayGame(ctx context.Context, userID *uuid.UUID, gameID uuid.UUID) error {
	g, err := queries().GetGameByID(ctx, gameID)
	if err != nil {
		return obj.ErrNotFound("game not found")
	}

	// Public games are accessible to everyone
	if g.Public {
		return nil
	}

	// Non-public games require ownership
	if userID == nil {
		return obj.ErrUnauthorized("access denied: authentication required")
	}
	if !g.CreatedBy.Valid || g.CreatedBy.UUID != *userID {
		return obj.ErrForbidden("access denied: not the owner of this game")
	}

	return nil
}

// GetGames returns games based on filters. If userID is provided, returns user's games.
// If PublicOnly filter is set, returns only public games.
// If Search is provided, filters games by name (case-insensitive).
// If SortField/SortDir are provided, results are sorted accordingly.
// Filter can be: all, own, public (organization and favorites fall back to all).
func GetGames(ctx context.Context, userID *uuid.UUID, filters *GetGamesFilters) ([]obj.Game, error) {
	var dbGames []db.Game
	var err error

	searchQuery := ""
	sortField := ""
	sortDir := "desc"
	filterType := "all"
	if filters != nil {
		searchQuery = filters.Search
		sortField = filters.SortField
		if filters.SortDir != "" {
			sortDir = filters.SortDir
		}
		if filters.Filter != "" {
			filterType = filters.Filter
		}
	}

	// Handle legacy PublicOnly flag
	if filters != nil && filters.PublicOnly {
		filterType = "public"
	}

	switch filterType {
	case "public":
		dbGames, err = getPublicGames(ctx, searchQuery, sortField, sortDir)
	case "own":
		if userID == nil {
			return nil, obj.ErrValidation("must provide userID for own filter")
		}
		dbGames, err = getOwnGames(ctx, *userID, searchQuery, sortField, sortDir)
	case "all", "organization", "favorites":
		// organization and favorites fall back to all for now
		if userID != nil {
			dbGames, err = getGamesVisibleToUser(ctx, *userID, searchQuery, sortField, sortDir)
		} else {
			// Unauthenticated users can only see public games
			dbGames, err = getPublicGames(ctx, searchQuery, sortField, sortDir)
		}
	default:
		if userID != nil {
			dbGames, err = getGamesVisibleToUser(ctx, *userID, searchQuery, sortField, sortDir)
		} else {
			return nil, obj.ErrValidation("must provide userID or valid filter")
		}
	}

	if err != nil {
		return nil, obj.ErrServerError("failed to get games")
	}

	result := make([]obj.Game, 0, len(dbGames))
	for _, g := range dbGames {
		game, err := dbGameToObj(ctx, g)
		if err != nil {
			return nil, err
		}
		result = append(result, *game)
	}
	return result, nil
}

// getPublicGames fetches public games with optional search and sorting
func getPublicGames(ctx context.Context, search, sortField, sortDir string) ([]db.Game, error) {
	searchParam := sql.NullString{String: search, Valid: search != ""}

	if search != "" {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				return queries().SearchPublicGamesSortedByName(ctx, searchParam)
			}
			return queries().SearchPublicGamesSortedByNameDesc(ctx, searchParam)
		case "createdAt":
			if sortDir == "asc" {
				return queries().SearchPublicGamesSortedByCreatedAt(ctx, searchParam)
			}
			return queries().SearchPublicGames(ctx, searchParam) // default createdAt desc
		case "modifiedAt":
			if sortDir == "asc" {
				return queries().SearchPublicGamesSortedByModifiedAtAsc(ctx, searchParam)
			}
			return queries().SearchPublicGamesSortedByModifiedAt(ctx, searchParam)
		default:
			return queries().SearchPublicGames(ctx, searchParam)
		}
	}

	switch sortField {
	case "name":
		if sortDir == "asc" {
			return queries().GetPublicGamesSortedByName(ctx)
		}
		return queries().GetPublicGamesSortedByNameDesc(ctx)
	case "createdAt":
		if sortDir == "asc" {
			return queries().GetPublicGamesSortedByCreatedAt(ctx)
		}
		return queries().GetPublicGames(ctx) // default createdAt desc
	case "modifiedAt":
		if sortDir == "asc" {
			return queries().GetPublicGamesSortedByModifiedAtAsc(ctx)
		}
		return queries().GetPublicGamesSortedByModifiedAt(ctx)
	default:
		return queries().GetPublicGames(ctx)
	}
}

// getOwnGames fetches games owned by user with optional search and sorting
func getOwnGames(ctx context.Context, userID uuid.UUID, search, sortField, sortDir string) ([]db.Game, error) {
	userParam := uuid.NullUUID{UUID: userID, Valid: true}
	searchStr := sql.NullString{String: search, Valid: search != ""}

	if search != "" {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				return queries().SearchOwnGamesSortedByName(ctx, db.SearchOwnGamesSortedByNameParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchOwnGamesSortedByNameDesc(ctx, db.SearchOwnGamesSortedByNameDescParams{CreatedBy: userParam, Column2: searchStr})
		case "createdAt":
			if sortDir == "asc" {
				return queries().SearchOwnGamesSortedByCreatedAt(ctx, db.SearchOwnGamesSortedByCreatedAtParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchOwnGames(ctx, db.SearchOwnGamesParams{CreatedBy: userParam, Column2: searchStr})
		case "modifiedAt":
			if sortDir == "asc" {
				return queries().SearchOwnGamesSortedByModifiedAtAsc(ctx, db.SearchOwnGamesSortedByModifiedAtAscParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchOwnGamesSortedByModifiedAt(ctx, db.SearchOwnGamesSortedByModifiedAtParams{CreatedBy: userParam, Column2: searchStr})
		default:
			return queries().SearchOwnGames(ctx, db.SearchOwnGamesParams{CreatedBy: userParam, Column2: searchStr})
		}
	}

	switch sortField {
	case "name":
		if sortDir == "asc" {
			return queries().GetOwnGamesSortedByName(ctx, userParam)
		}
		return queries().GetOwnGamesSortedByNameDesc(ctx, userParam)
	case "createdAt":
		if sortDir == "asc" {
			return queries().GetOwnGamesSortedByCreatedAt(ctx, userParam)
		}
		return queries().GetOwnGames(ctx, userParam)
	case "modifiedAt":
		if sortDir == "asc" {
			return queries().GetOwnGamesSortedByModifiedAtAsc(ctx, userParam)
		}
		return queries().GetOwnGamesSortedByModifiedAt(ctx, userParam)
	default:
		return queries().GetOwnGames(ctx, userParam)
	}
}

// getGamesVisibleToUser fetches games visible to user with optional search and sorting
// Also includes games from the user's workshop (if they belong to one)
func getGamesVisibleToUser(ctx context.Context, userID uuid.UUID, search, sortField, sortDir string) ([]db.Game, error) {
	userParam := uuid.NullUUID{UUID: userID, Valid: true}
	searchStr := sql.NullString{String: search, Valid: search != ""}

	// Get user's workshop ID (if any) to include workshop games
	var workshopParam uuid.NullUUID
	user, err := GetUserByID(ctx, userID)
	if err == nil && user.Role != nil && user.Role.Workshop != nil {
		workshopParam = uuid.NullUUID{UUID: user.Role.Workshop.ID, Valid: true}
	}

	if search != "" {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByName(ctx, db.SearchGamesVisibleToUserSortedByNameParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			}
			return queries().SearchGamesVisibleToUserSortedByNameDesc(ctx, db.SearchGamesVisibleToUserSortedByNameDescParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
		case "createdAt":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByCreatedAt(ctx, db.SearchGamesVisibleToUserSortedByCreatedAtParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			}
			return queries().SearchGamesVisibleToUser(ctx, db.SearchGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
		case "modifiedAt":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByModifiedAtAsc(ctx, db.SearchGamesVisibleToUserSortedByModifiedAtAscParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			}
			return queries().SearchGamesVisibleToUserSortedByModifiedAt(ctx, db.SearchGamesVisibleToUserSortedByModifiedAtParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
		default:
			return queries().SearchGamesVisibleToUser(ctx, db.SearchGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
		}
	}

	switch sortField {
	case "name":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByName(ctx, db.GetGamesVisibleToUserSortedByNameParams{CreatedBy: userParam, WorkshopID: workshopParam})
		}
		return queries().GetGamesVisibleToUserSortedByNameDesc(ctx, db.GetGamesVisibleToUserSortedByNameDescParams{CreatedBy: userParam, WorkshopID: workshopParam})
	case "createdAt":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByCreatedAt(ctx, db.GetGamesVisibleToUserSortedByCreatedAtParams{CreatedBy: userParam, WorkshopID: workshopParam})
		}
		return queries().GetGamesVisibleToUser(ctx, db.GetGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam})
	case "modifiedAt":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByModifiedAtAsc(ctx, db.GetGamesVisibleToUserSortedByModifiedAtAscParams{CreatedBy: userParam, WorkshopID: workshopParam})
		}
		return queries().GetGamesVisibleToUserSortedByModifiedAt(ctx, db.GetGamesVisibleToUserSortedByModifiedAtParams{CreatedBy: userParam, WorkshopID: workshopParam})
	default:
		return queries().GetGamesVisibleToUser(ctx, db.GetGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam})
	}
}

// GetGameByID gets a game by ID. Verifies access based on user permissions.
func GetGameByID(ctx context.Context, userID *uuid.UUID, gameID uuid.UUID) (*obj.Game, error) {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	// Always check permissions (anonymous users can access public games)
	checkUserID := uuid.Nil
	if userID != nil {
		checkUserID = *userID
	}
	if err := canAccessGame(ctx, checkUserID, OpRead, game, nil); err != nil {
		return nil, err
	}

	return game, nil
}

// GetGameByToken gets a game by its private share hash (token).
// This needs no access check, because games with such a token are public by definition
func GetGameByToken(ctx context.Context, token string) (*obj.Game, error) {
	g, err := queries().GetGameByPrivateShareHash(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		return nil, obj.ErrNotFound("game not found")
	}
	return dbGameToObj(ctx, g)
}

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

	arg := db.CreateGameParams{
		ID:                       game.ID,
		CreatedBy:                uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:                now,
		ModifiedBy:               uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:               now,
		Name:                     game.Name,
		Description:              game.Description,
		Icon:                     game.Icon,
		WorkshopID:               uuidPtrToNullUUID(game.WorkshopID),
		Public:                   game.Public,
		PublicSponsoredApiKeyID:  uuidPtrToNullUUID(game.PublicSponsoredApiKeyID),
		PrivateShareHash:         sql.NullString{String: functional.Deref(game.PrivateShareHash, ""), Valid: game.PrivateShareHash != nil},
		PrivateSponsoredApiKeyID: uuidPtrToNullUUID(game.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    game.SystemMessageScenario,
		SystemMessageGameStart:   game.SystemMessageGameStart,
		ImageStyle:               game.ImageStyle,
		Css:                      game.CSS,
		StatusFields:             game.StatusFields,
		FirstMessage:             sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:              sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:               game.FirstImage,
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
	privateShareHash := sql.NullString{String: functional.Deref(game.PrivateShareHash, ""), Valid: game.PrivateShareHash != nil}
	if !privateShareHash.Valid || privateShareHash.String == "" {
		// Keep existing hash or generate new one
		if existingGameRaw.PrivateShareHash.Valid && existingGameRaw.PrivateShareHash.String != "" {
			privateShareHash = existingGameRaw.PrivateShareHash
		} else {
			hash, _ := functional.GenerateSecureToken(20)
			privateShareHash = sql.NullString{String: hash, Valid: true}
		}
	}

	arg := db.UpdateGameParams{
		ID:                       game.ID,
		CreatedBy:                existingGameRaw.CreatedBy,
		CreatedAt:                existingGameRaw.CreatedAt,
		ModifiedBy:               uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:               now,
		Name:                     game.Name,
		Description:              game.Description,
		Icon:                     game.Icon,
		Public:                   game.Public,
		PublicSponsoredApiKeyID:  uuidPtrToNullUUID(game.PublicSponsoredApiKeyID),
		PrivateShareHash:         privateShareHash,
		PrivateSponsoredApiKeyID: uuidPtrToNullUUID(game.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    game.SystemMessageScenario,
		SystemMessageGameStart:   game.SystemMessageGameStart,
		ImageStyle:               game.ImageStyle,
		Css:                      game.CSS,
		StatusFields:             game.StatusFields,
		FirstMessage:             sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:              sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:               game.FirstImage,
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

// CreateGameSession creates a new game session with minimal required parameters.
// The function loads game details and constructs the session object internally.
// Parameters:
// - userID: the user creating the session
// - gameID: the game to play
// - apiKeyID: the API key to use (defines platform)
// - aiModel: the AI model to use
// - workshopID: optional workshop context
// - theme: optional visual theme for the game player UI
func CreateGameSession(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, apiKeyID uuid.UUID, aiModel string, workshopID *uuid.UUID, theme *obj.GameTheme) (*obj.GameSession, error) {
	// Validate workshop access and game permissions
	if err := canAccessGameSession(ctx, userID, OpCreate, nil, gameID, workshopID); err != nil {
		return nil, err
	}

	// Load game to get details
	game, err := queries().GetGameByID(ctx, gameID)
	if err != nil {
		return nil, obj.ErrNotFound("game not found")
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

	now := time.Now()
	arg := db.CreateGameSessionParams{
		CreatedBy:    uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:    now,
		ModifiedBy:   uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:   now,
		GameID:       gameID,
		UserID:       userID,
		WorkshopID:   uuidPtrToNullUUID(workshopID),
		ApiKeyID:     uuid.NullUUID{UUID: apiKeyID, Valid: true},
		AiPlatform:   apiKey.Platform,
		AiModel:      aiModel,
		AiSession:    []byte("{}"), // Empty JSON object as initial state
		ImageStyle:   game.ImageStyle,
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
		UserID:          result.UserID,
		WorkshopID:      nullUUIDToPtr(result.WorkshopID),
		ApiKeyID:        nullUUIDToPtr(result.ApiKeyID),
		AiPlatform:      result.AiPlatform,
		AiModel:         result.AiModel,
		AiSession:       string(result.AiSession),
		ImageStyle:      result.ImageStyle,
		StatusFields:    result.StatusFields,
		Theme:           theme,
	}, nil
}

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
		CreatedBy:     uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:     now,
		ModifiedBy:    uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:    now,
		GameSessionID: msg.GameSessionID,
		Type:          msg.Type,
		Message:       msg.Message,
		Status:        statusJSON,
		ImagePrompt:   sql.NullString{String: functional.Deref(msg.ImagePrompt, ""), Valid: msg.ImagePrompt != nil},
		Image:         msg.Image,
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

	arg := db.UpdateGameSessionMessageParams{
		ID:            msg.ID,
		CreatedBy:     uuid.NullUUID{},
		CreatedAt:     time.Time{},
		ModifiedBy:    uuid.NullUUID{},
		ModifiedAt:    now,
		GameSessionID: msg.GameSessionID,
		Type:          msg.Type,
		Message:       msg.Message,
		Status:        statusJSON,
		ImagePrompt:   sql.NullString{String: functional.Deref(msg.ImagePrompt, ""), Valid: msg.ImagePrompt != nil},
		Image:         msg.Image,
	}

	_, err = queries().UpdateGameSessionMessage(ctx, arg)
	if err != nil {
		return obj.ErrServerError("failed to update session message")
	}

	return nil
}

// UpdateGameSessionAiSession updates the AI session state for a game session
func UpdateGameSessionAiSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, aiSession string) error {
	// Verify session ownership
	sessionObj, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	_, err = queries().UpdateGameSessionAiSession(ctx, db.UpdateGameSessionAiSessionParams{
		ID:        sessionID,
		AiSession: []byte(aiSession),
	})
	if err != nil {
		return obj.ErrServerError("failed to update session AI state")
	}
	return nil
}

// UpdateGameSessionTheme updates the visual theme for a game session
func UpdateGameSessionTheme(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, theme *obj.GameTheme) error {
	// Verify session ownership
	sessionObj, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := canAccessGameSession(ctx, userID, OpUpdate, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return err
	}

	var themeJSON pqtype.NullRawMessage
	if theme != nil {
		themeBytes, err := json.Marshal(theme)
		if err != nil {
			return obj.ErrServerError("failed to marshal theme")
		}
		themeJSON = pqtype.NullRawMessage{RawMessage: themeBytes, Valid: true}
	}

	err = queries().UpdateGameSessionTheme(ctx, db.UpdateGameSessionThemeParams{
		ID:    sessionID,
		Theme: themeJSON,
	})
	if err != nil {
		return obj.ErrServerError("failed to update session theme")
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
	}

	// Parse theme from JSON if present
	if s.Theme.Valid && len(s.Theme.RawMessage) > 0 {
		var theme obj.GameTheme
		if err := json.Unmarshal(s.Theme.RawMessage, &theme); err == nil {
			session.Theme = &theme
		}
	}

	// Load game info
	game, err := queries().GetGameByID(ctx, s.GameID)
	if err == nil {
		session.GameName = game.Name
		session.GameDescription = game.Description
	}

	// Load API key (if present - may be null if the key was deleted)
	if s.ApiKeyID.Valid {
		key, err := queries().GetApiKeyByID(ctx, s.ApiKeyID.UUID)
		if err == nil {
			session.ApiKey = &obj.ApiKey{
				ID:       key.ID,
				UserID:   key.UserID,
				Name:     key.Name,
				Platform: key.Platform,
				Key:      key.Key,
			}
		}
		// If key not found, leave ApiKey as nil - frontend will prompt for a new one
	}

	return session, nil
}

// UpdateGameSessionApiKey updates the API key for a session (used when resuming a session whose key was deleted)
func UpdateGameSessionApiKey(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, shareID uuid.UUID, model string) (*obj.GameSession, error) {
	// Load and verify session ownership
	session, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("session not found")
	}
	if session.UserID != userID {
		return nil, obj.ErrForbidden("not the owner of this session")
	}

	// Resolve the API key share
	share, err := GetApiKeyShareByID(ctx, userID, shareID)
	if err != nil {
		return nil, obj.ErrValidation("invalid API key share")
	}
	if share.ApiKey == nil {
		return nil, obj.ErrValidation("API key not found in share")
	}

	// Determine AI model - use provided or fall back to platform default
	aiModel := model
	if aiModel == "" {
		aiModel = obj.AiModelBalanced
	}

	// Update the session
	result, err := queries().UpdateGameSessionApiKey(ctx, db.UpdateGameSessionApiKeyParams{
		ID:         sessionID,
		ApiKeyID:   uuid.NullUUID{UUID: share.ApiKey.ID, Valid: true},
		AiPlatform: share.ApiKey.Platform,
		AiModel:    aiModel,
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to update session: " + err.Error())
	}

	return &obj.GameSession{
		ID:         result.ID,
		GameID:     result.GameID,
		UserID:     result.UserID,
		ApiKeyID:   nullUUIDToPtr(result.ApiKeyID),
		AiPlatform: result.AiPlatform,
		AiModel:    result.AiModel,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// ClearGameSessionApiKey clears the API key reference from a session
// Used when an API key becomes invalid (billing not active, key revoked, etc.)
func ClearGameSessionApiKey(ctx context.Context, sessionID uuid.UUID) error {
	return queries().ClearGameSessionApiKeyByID(ctx, sessionID)
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

// GetGameSessionMessageByID returns a message by its ID (requires read access to session)
func GetGameSessionMessageByID(ctx context.Context, userID uuid.UUID, messageID uuid.UUID) (*obj.GameSessionMessage, error) {
	m, err := queries().GetGameSessionMessageByID(ctx, messageID)
	if err != nil {
		return nil, obj.ErrNotFound("message not found")
	}

	// Check if user has read access to the session
	sessionObj, err := loadSessionByID(ctx, m.GameSessionID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGameSession(ctx, userID, OpRead, sessionObj, sessionObj.GameID, sessionObj.WorkshopID); err != nil {
		return nil, err
	}

	msg := &obj.GameSessionMessage{
		ID:            m.ID,
		GameSessionID: m.GameSessionID,
		Seq:           int(m.Seq),
		Type:          m.Type,
		Message:       m.Message,
		Image:         m.Image,
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

	// Set image prompt
	if m.ImagePrompt.Valid {
		msg.ImagePrompt = &m.ImagePrompt.String
	}

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

	// Set image prompt
	if m.ImagePrompt.Valid {
		msg.ImagePrompt = &m.ImagePrompt.String
	}

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

		// Set image prompt
		if m.ImagePrompt.Valid {
			msg.ImagePrompt = &m.ImagePrompt.String
		}

		result = append(result, msg)
	}

	return result, nil
}

// DeleteGameSessionMessage deletes a single message by ID.
// Used to clean up placeholder messages when AI actions fail.
func DeleteGameSessionMessage(ctx context.Context, messageID uuid.UUID) error {
	return queries().DeleteGameSessionMessage(ctx, messageID)
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

// DeleteEmptyGameSession deletes a newly created session and its messages.
// Used to clean up sessions that failed during initial action (has streaming placeholder but no real content).
// No permission check - called internally after creation failure.
func DeleteEmptyGameSession(ctx context.Context, sessionID uuid.UUID) error {
	// Delete messages first (the streaming placeholder)
	if err := queries().DeleteNewlyCreatedGameSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session messages: %w", err)
	}
	// Then delete the session
	return queries().DeleteEmptyGameSession(ctx, sessionID)
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

// dbGameToObj converts a sqlc Game to obj.Game, including tags
func dbGameToObj(ctx context.Context, g db.Game) (*obj.Game, error) {
	// Get tags for this game
	dbTags, err := queries().GetGameTagsByGameID(ctx, g.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game tags: %w", err)
	}

	tags := make([]obj.GameTag, 0, len(dbTags))
	for _, t := range dbTags {
		tags = append(tags, obj.GameTag{
			ID: t.ID,
			Meta: obj.Meta{
				CreatedBy:  t.CreatedBy,
				CreatedAt:  &t.CreatedAt,
				ModifiedBy: t.ModifiedBy,
				ModifiedAt: &t.ModifiedAt,
			},
			GameID: t.GameID,
			Tag:    t.Tag,
		})
	}

	game := &obj.Game{
		ID: g.ID,
		Meta: obj.Meta{
			CreatedBy:  g.CreatedBy,
			CreatedAt:  &g.CreatedAt,
			ModifiedBy: g.ModifiedBy,
			ModifiedAt: &g.ModifiedAt,
		},
		Name:                     g.Name,
		Description:              g.Description,
		Icon:                     g.Icon,
		Public:                   g.Public,
		PublicSponsoredApiKeyID:  nullUUIDToPtr(g.PublicSponsoredApiKeyID),
		PrivateShareHash:         nullStringToPtr(g.PrivateShareHash),
		PrivateSponsoredApiKeyID: nullUUIDToPtr(g.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    g.SystemMessageScenario,
		SystemMessageGameStart:   g.SystemMessageGameStart,
		ImageStyle:               g.ImageStyle,
		CSS:                      g.Css,
		StatusFields:             g.StatusFields,
		FirstMessage:             nullStringToPtr(g.FirstMessage),
		FirstStatus:              nullStringToPtr(g.FirstStatus),
		FirstImage:               g.FirstImage,
		Tags:                     tags,
		PlayCount:                int(g.PlayCount),
		CloneCount:               int(g.CloneCount),
		OriginallyCreatedBy:      nullUUIDToPtr(g.OriginallyCreatedBy),
	}

	// Populate creator info from CreatedBy
	if g.CreatedBy.Valid {
		game.CreatorID = &g.CreatedBy.UUID
		// Fetch creator name
		if user, err := GetUserByID(ctx, g.CreatedBy.UUID); err == nil && user != nil {
			game.CreatorName = &user.Name
		}
	}

	// Populate workshop ID
	if g.WorkshopID.Valid {
		game.WorkshopID = &g.WorkshopID.UUID
	}

	// Populate original creator info if cloned
	if g.OriginallyCreatedBy.Valid {
		game.OriginalCreatorID = &g.OriginallyCreatedBy.UUID
		if user, err := GetUserByID(ctx, g.OriginallyCreatedBy.UUID); err == nil && user != nil {
			game.OriginalCreatorName = &user.Name
		}
	}

	return game, nil
}

// loadGameByID loads a game from DB and converts it to obj.Game
func loadGameByID(ctx context.Context, gameID uuid.UUID) (*obj.Game, error) {
	game, _, err := loadGameByIDWithRaw(ctx, gameID)
	return game, err
}

// loadGameByIDWithRaw loads a game from DB and returns both the parsed object and raw DB row
func loadGameByIDWithRaw(ctx context.Context, gameID uuid.UUID) (*obj.Game, *db.Game, error) {
	g, err := queries().GetGameByID(ctx, gameID)
	if err != nil {
		return nil, nil, obj.ErrNotFound("game not found")
	}
	game, err := dbGameToObj(ctx, g)
	if err != nil {
		return nil, nil, err
	}
	return game, &g, nil
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

// IncrementGameCloneCount increments the clone count for a game
func IncrementGameCloneCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGameCloneCount(ctx, gameID)
}

// Helper functions for converting between sql.Null* types and pointers

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func nullUUIDToPtr(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

func uuidPtrToNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}
