package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
		return fmt.Errorf("game not found: %w", err)
	}

	// Public games are accessible to everyone
	if g.Public {
		return nil
	}

	// Non-public games require ownership
	if userID == nil {
		return errors.New("access denied: authentication required")
	}
	if !g.CreatedBy.Valid || g.CreatedBy.UUID != *userID {
		return errors.New("access denied: not the owner of this game")
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
			return nil, errors.New("must provide userID for own filter")
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
			return nil, errors.New("must provide userID or valid filter")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
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
		case "playCount":
			if sortDir == "asc" {
				return queries().SearchPublicGamesSortedByPlayCountAsc(ctx, searchParam)
			}
			return queries().SearchPublicGamesSortedByPlayCount(ctx, searchParam)
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
	case "playCount":
		if sortDir == "asc" {
			return queries().GetPublicGamesSortedByPlayCountAsc(ctx)
		}
		return queries().GetPublicGamesSortedByPlayCount(ctx)
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
		case "playCount":
			if sortDir == "asc" {
				return queries().SearchOwnGamesSortedByPlayCountAsc(ctx, db.SearchOwnGamesSortedByPlayCountAscParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchOwnGamesSortedByPlayCount(ctx, db.SearchOwnGamesSortedByPlayCountParams{CreatedBy: userParam, Column2: searchStr})
		case "visibility":
			if sortDir == "asc" {
				return queries().SearchOwnGamesSortedByVisibilityAsc(ctx, db.SearchOwnGamesSortedByVisibilityAscParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchOwnGamesSortedByVisibility(ctx, db.SearchOwnGamesSortedByVisibilityParams{CreatedBy: userParam, Column2: searchStr})
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
	case "playCount":
		if sortDir == "asc" {
			return queries().GetOwnGamesSortedByPlayCountAsc(ctx, userParam)
		}
		return queries().GetOwnGamesSortedByPlayCount(ctx, userParam)
	case "visibility":
		if sortDir == "asc" {
			return queries().GetOwnGamesSortedByVisibilityAsc(ctx, userParam)
		}
		return queries().GetOwnGamesSortedByVisibility(ctx, userParam)
	default:
		return queries().GetOwnGames(ctx, userParam)
	}
}

// getGamesVisibleToUser fetches games visible to user with optional search and sorting
func getGamesVisibleToUser(ctx context.Context, userID uuid.UUID, search, sortField, sortDir string) ([]db.Game, error) {
	userParam := uuid.NullUUID{UUID: userID, Valid: true}
	searchStr := sql.NullString{String: search, Valid: search != ""}

	if search != "" {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByName(ctx, db.SearchGamesVisibleToUserSortedByNameParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchGamesVisibleToUserSortedByNameDesc(ctx, db.SearchGamesVisibleToUserSortedByNameDescParams{CreatedBy: userParam, Column2: searchStr})
		case "createdAt":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByCreatedAt(ctx, db.SearchGamesVisibleToUserSortedByCreatedAtParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchGamesVisibleToUser(ctx, db.SearchGamesVisibleToUserParams{CreatedBy: userParam, Column2: searchStr})
		case "modifiedAt":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByModifiedAtAsc(ctx, db.SearchGamesVisibleToUserSortedByModifiedAtAscParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchGamesVisibleToUserSortedByModifiedAt(ctx, db.SearchGamesVisibleToUserSortedByModifiedAtParams{CreatedBy: userParam, Column2: searchStr})
		case "playCount":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByPlayCountAsc(ctx, db.SearchGamesVisibleToUserSortedByPlayCountAscParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchGamesVisibleToUserSortedByPlayCount(ctx, db.SearchGamesVisibleToUserSortedByPlayCountParams{CreatedBy: userParam, Column2: searchStr})
		case "creator":
			if sortDir == "asc" {
				return queries().SearchGamesVisibleToUserSortedByCreator(ctx, db.SearchGamesVisibleToUserSortedByCreatorParams{CreatedBy: userParam, Column2: searchStr})
			}
			return queries().SearchGamesVisibleToUserSortedByCreatorDesc(ctx, db.SearchGamesVisibleToUserSortedByCreatorDescParams{CreatedBy: userParam, Column2: searchStr})
		default:
			return queries().SearchGamesVisibleToUser(ctx, db.SearchGamesVisibleToUserParams{CreatedBy: userParam, Column2: searchStr})
		}
	}

	switch sortField {
	case "name":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByName(ctx, userParam)
		}
		return queries().GetGamesVisibleToUserSortedByNameDesc(ctx, userParam)
	case "createdAt":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByCreatedAt(ctx, userParam)
		}
		return queries().GetGamesVisibleToUser(ctx, userParam)
	case "modifiedAt":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByModifiedAtAsc(ctx, userParam)
		}
		return queries().GetGamesVisibleToUserSortedByModifiedAt(ctx, userParam)
	case "playCount":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByPlayCountAsc(ctx, userParam)
		}
		return queries().GetGamesVisibleToUserSortedByPlayCount(ctx, userParam)
	case "creator":
		if sortDir == "asc" {
			return queries().GetGamesVisibleToUserSortedByCreator(ctx, userParam)
		}
		return queries().GetGamesVisibleToUserSortedByCreatorDesc(ctx, userParam)
	default:
		return queries().GetGamesVisibleToUser(ctx, userParam)
	}
}

// GetGameByID gets a game by ID. If userID is provided, verifies ownership.
func GetGameByID(ctx context.Context, userID *uuid.UUID, gameID uuid.UUID) (*obj.Game, error) {
	g, err := queries().GetGameByID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("game not found: %w", err)
	}

	// If userID provided, verify ownership (unless game is public)
	if userID != nil && !g.Public {
		if !g.CreatedBy.Valid || g.CreatedBy.UUID != *userID {
			return nil, errors.New("access denied: not the owner of this game")
		}
	}

	return dbGameToObj(ctx, g)
}

// GetGameByToken gets a game by its private share hash (token).
func GetGameByToken(ctx context.Context, token string) (*obj.Game, error) {
	g, err := queries().GetGameByPrivateShareHash(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("game not found: %w", err)
	}
	return dbGameToObj(ctx, g)
}

// DeleteGame deletes a game. userID must be the owner.
func DeleteGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	// Verify ownership
	g, err := queries().GetGameByID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}
	if !g.CreatedBy.Valid || g.CreatedBy.UUID != userID {
		return errors.New("access denied: not the owner of this game")
	}

	return queries().DeleteGame(ctx, gameID)
}

// CreateGame creates a new game. userID is set as the owner (createdBy).
func CreateGame(ctx context.Context, userID uuid.UUID, game *obj.Game) error {
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
		Public:                   game.Public,
		PublicSponsoredApiKeyID:  uuidPtrToNullUUID(game.PublicSponsoredApiKeyID),
		PrivateShareHash:         sql.NullString{String: ptrToString(game.PrivateShareHash), Valid: game.PrivateShareHash != nil},
		PrivateSponsoredApiKeyID: uuidPtrToNullUUID(game.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    game.SystemMessageScenario,
		SystemMessageGameStart:   game.SystemMessageGameStart,
		ImageStyle:               game.ImageStyle,
		Css:                      game.CSS,
		StatusFields:             game.StatusFields,
		FirstMessage:             sql.NullString{String: ptrToString(game.FirstMessage), Valid: game.FirstMessage != nil},
		FirstStatus:              sql.NullString{String: ptrToString(game.FirstStatus), Valid: game.FirstStatus != nil},
		FirstImage:               game.FirstImage,
		OriginallyCreatedBy:      uuidPtrToNullUUID(game.OriginallyCreatedBy),
	}

	// Generate private share hash if not provided
	if !arg.PrivateShareHash.Valid || arg.PrivateShareHash.String == "" {
		arg.PrivateShareHash = sql.NullString{String: randomHash(), Valid: true}
	}

	_, err := queries().CreateGame(ctx, arg)
	return err
}

// UpdateGame updates an existing game. userID must be the owner.
func UpdateGame(ctx context.Context, userID uuid.UUID, game *obj.Game) error {
	// Verify ownership
	existing, err := queries().GetGameByID(ctx, game.ID)
	if err != nil {
		return fmt.Errorf("game not found: %w", err)
	}
	if !existing.CreatedBy.Valid || existing.CreatedBy.UUID != userID {
		return errors.New("access denied: not the owner of this game")
	}

	now := time.Now()
	privateShareHash := sql.NullString{String: ptrToString(game.PrivateShareHash), Valid: game.PrivateShareHash != nil}
	if !privateShareHash.Valid || privateShareHash.String == "" {
		// Keep existing hash or generate new one
		if existing.PrivateShareHash.Valid && existing.PrivateShareHash.String != "" {
			privateShareHash = existing.PrivateShareHash
		} else {
			privateShareHash = sql.NullString{String: randomHash(), Valid: true}
		}
	}

	arg := db.UpdateGameParams{
		ID:                       game.ID,
		CreatedBy:                existing.CreatedBy,
		CreatedAt:                existing.CreatedAt,
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
		FirstMessage:             sql.NullString{String: ptrToString(game.FirstMessage), Valid: game.FirstMessage != nil},
		FirstStatus:              sql.NullString{String: ptrToString(game.FirstStatus), Valid: game.FirstStatus != nil},
		FirstImage:               game.FirstImage,
		OriginallyCreatedBy:      existing.OriginallyCreatedBy, // Preserve original creator
	}

	_, err = queries().UpdateGame(ctx, arg)
	return err
}

// UpdateGameYaml updates a game from YAML content. userID must be the owner.
func UpdateGameYaml(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, yamlContent string) error {
	log.Debug("UpdateGameYaml: starting", "user_id", userID, "game_id", gameID)

	// Get existing game first
	existing, err := GetGameByID(ctx, &userID, gameID)
	if err != nil {
		log.Debug("UpdateGameYaml: GetGameByID failed", "error", err)
		return fmt.Errorf("game not found: %w", err)
	}
	log.Debug("UpdateGameYaml: existing game loaded", "name", existing.Name)

	// Parse YAML into a game object
	var incoming obj.Game
	if err := yaml.Unmarshal([]byte(yamlContent), &incoming); err != nil {
		log.Debug("UpdateGameYaml: YAML unmarshal failed", "error", err)
		return fmt.Errorf("invalid YAML: %w", err)
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

// CreateGameSession persists a game session to the database and returns the created session with DB-generated ID
func CreateGameSession(ctx context.Context, session *obj.GameSession) (*obj.GameSession, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}
	now := time.Now()

	// Serialize theme to JSON if present
	var themeJSON pqtype.NullRawMessage
	if session.Theme != nil {
		themeBytes, err := json.Marshal(session.Theme)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize theme: %w", err)
		}
		themeJSON = pqtype.NullRawMessage{RawMessage: themeBytes, Valid: true}
	}

	arg := db.CreateGameSessionParams{
		CreatedBy:    uuid.NullUUID{UUID: session.UserID, Valid: true},
		CreatedAt:    now,
		ModifiedBy:   uuid.NullUUID{UUID: session.UserID, Valid: true},
		ModifiedAt:   now,
		GameID:       session.GameID,
		UserID:       session.UserID,
		ApiKeyID:     session.ApiKeyID,
		AiPlatform:   session.AiPlatform,
		AiModel:      session.AiModel,
		AiSession:    []byte(session.AiSession),
		ImageStyle:   session.ImageStyle,
		StatusFields: session.StatusFields,
		Theme:        themeJSON,
	}

	result, err := queries().CreateGameSession(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	session.ID = result.ID
	session.Meta.CreatedAt = &result.CreatedAt
	session.Meta.ModifiedAt = &result.ModifiedAt

	return session, nil
}

// CreateGameSessionMessage adds a message to a game session with auto-incremented seq
func CreateGameSessionMessage(ctx context.Context, userID uuid.UUID, msg obj.GameSessionMessage) (*obj.GameSessionMessage, error) {
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
		ImagePrompt:   sql.NullString{String: ptrToString(msg.ImagePrompt), Valid: msg.ImagePrompt != nil},
		Image:         msg.Image,
	}

	result, err := queries().CreateGameSessionMessage(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to create session message: %w", err)
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
func UpdateGameSessionMessage(ctx context.Context, msg obj.GameSessionMessage) error {
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
		ImagePrompt:   sql.NullString{String: ptrToString(msg.ImagePrompt), Valid: msg.ImagePrompt != nil},
		Image:         msg.Image,
	}

	_, err := queries().UpdateGameSessionMessage(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to update session message: %w", err)
	}

	return nil
}

// UpdateGameSessionAiSession updates the AI session state for a game session
func UpdateGameSessionAiSession(ctx context.Context, sessionID uuid.UUID, aiSession string) error {
	_, err := queries().UpdateGameSessionAiSession(ctx, db.UpdateGameSessionAiSessionParams{
		ID:        sessionID,
		AiSession: []byte(aiSession),
	})
	if err != nil {
		return fmt.Errorf("failed to update session AI state: %w", err)
	}
	return nil
}

// UpdateGameSessionMessageImage updates only the image field of a message
func UpdateGameSessionMessageImage(ctx context.Context, messageID uuid.UUID, image []byte) error {
	_, err := queries().UpdateGameSessionMessageImage(ctx, db.UpdateGameSessionMessageImageParams{
		ID:    messageID,
		Image: image,
	})
	if err != nil {
		return fmt.Errorf("failed to update message image: %w", err)
	}
	return nil
}

// GetGameSessionByID returns a single session by ID with its API key loaded
func GetGameSessionByID(ctx context.Context, userID *uuid.UUID, sessionID uuid.UUID) (*obj.GameSession, error) {
	s, err := queries().GetGameSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if userID != nil {
		if s.UserID != *userID {
			return nil, fmt.Errorf("failed to get session: access denied for user %s and session %s", userID.String(), s.ID.String())
		}
	}

	session := &obj.GameSession{
		ID:           s.ID,
		GameID:       s.GameID,
		UserID:       s.UserID,
		ApiKeyID:     s.ApiKeyID,
		AiPlatform:   s.AiPlatform,
		AiModel:      s.AiModel,
		AiSession:    string(s.AiSession),
		ImageStyle:   s.ImageStyle,
		StatusFields: s.StatusFields,
		Meta: obj.Meta{
			CreatedBy:  s.CreatedBy,
			CreatedAt:  &s.CreatedAt,
			ModifiedBy: s.ModifiedBy,
			ModifiedAt: &s.ModifiedAt,
		},
	}

	// Deserialize theme if present
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

	// Load API key
	key, err := queries().GetApiKeyByID(ctx, s.ApiKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key for session: %w", err)
	}
	session.ApiKey = &obj.ApiKey{
		ID:       key.ID,
		UserID:   key.UserID,
		Name:     key.Name,
		Platform: key.Platform,
		Key:      key.Key,
	}

	return session, nil
}

// GetGameSessionMessageByID returns a message by its ID
func GetGameSessionMessageByID(ctx context.Context, messageID uuid.UUID) (*obj.GameSessionMessage, error) {
	m, err := queries().GetGameSessionMessageByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
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

// GetLatestGameSessionMessage returns the most recent message for a session
func GetLatestGameSessionMessage(ctx context.Context, sessionID uuid.UUID) (*obj.GameSessionMessage, error) {
	m, err := queries().GetLatestGameSessionMessage(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest message: %w", err)
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

// GetAllGameSessionMessages returns all messages for a session ordered by sequence
func GetAllGameSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]obj.GameSessionMessage, error) {
	messages, err := queries().GetAllGameSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
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
func sessionRowToUserSession(id, gameID, userID, apiKeyID uuid.UUID, aiPlatform, aiModel string, aiSession []byte, imageStyle string, createdBy, modifiedBy uuid.NullUUID, createdAt, modifiedAt time.Time, gameName string) UserSessionWithGame {
	return UserSessionWithGame{
		GameSession: obj.GameSession{
			ID:         id,
			GameID:     gameID,
			UserID:     userID,
			ApiKeyID:   apiKeyID,
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
				return nil, fmt.Errorf("failed to get user sessions: %w", err)
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		case "model":
			rows, err := queries().SearchGameSessionsByUserIDSortByModel(ctx, db.SearchGameSessionsByUserIDSortByModelParams{UserID: userID, Column2: searchParam})
			if err != nil {
				return nil, fmt.Errorf("failed to get user sessions: %w", err)
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		default:
			rows, err := queries().SearchGameSessionsByUserID(ctx, db.SearchGameSessionsByUserIDParams{UserID: userID, Column2: searchParam})
			if err != nil {
				return nil, fmt.Errorf("failed to get user sessions: %w", err)
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
				return nil, fmt.Errorf("failed to get user sessions: %w", err)
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		case "model":
			rows, err := queries().GetGameSessionsByUserIDSortByModel(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to get user sessions: %w", err)
			}
			for _, s := range rows {
				sessions = append(sessions, sessionRowToUserSession(s.ID, s.GameID, s.UserID, s.ApiKeyID, s.AiPlatform, s.AiModel, s.AiSession, s.ImageStyle, s.CreatedBy, s.ModifiedBy, s.CreatedAt, s.ModifiedAt, s.GameName))
			}
		default:
			rows, err := queries().GetGameSessionsByUserID(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to get user sessions: %w", err)
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
	// Verify ownership
	session, err := queries().GetGameSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}
	if session.UserID != userID {
		return errors.New("access denied: not the owner of this session")
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
	// First delete all messages for sessions belonging to this user+game
	sessions, err := queries().GetGameSessionsByGameID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}
	for _, s := range sessions {
		if s.UserID == userID {
			if err := queries().DeleteGameSessionMessagesBySessionID(ctx, s.ID); err != nil {
				return fmt.Errorf("failed to delete session messages: %w", err)
			}
		}
	}

	// Then delete the sessions
	return queries().DeleteUserGameSessions(ctx, db.DeleteUserGameSessionsParams{
		UserID: userID,
		GameID: gameID,
	})
}

// GetGameSessionsByGameID returns all sessions for a game
func GetGameSessionsByGameID(ctx context.Context, gameID uuid.UUID) ([]obj.GameSession, error) {
	// TODO: we should consider user access rights here!
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
			ApiKeyID:   s.ApiKeyID,
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
		OriginallyCreatedBy:      nullUUIDToPtr(g.OriginallyCreatedBy),
		PlayCount:                int(g.PlayCount),
		CloneCount:               int(g.CloneCount),
	}

	// Populate creator info
	if g.CreatedBy.Valid {
		game.CreatorID = &g.CreatedBy.UUID
		creator, err := queries().GetUserByID(ctx, g.CreatedBy.UUID)
		if err == nil {
			game.CreatorName = &creator.Name
		}
	}

	// Populate original creator info if this is a cloned game
	if g.OriginallyCreatedBy.Valid {
		game.OriginalCreatorID = &g.OriginallyCreatedBy.UUID
		originalCreator, err := queries().GetUserByID(ctx, g.OriginallyCreatedBy.UUID)
		if err == nil {
			game.OriginalCreatorName = &originalCreator.Name
		}
	}

	return game, nil
}

// IncrementGamePlayCount increments the play_count for a game
func IncrementGamePlayCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGamePlayCount(ctx, gameID)
}

// IncrementGameCloneCount increments the clone_count for a game
func IncrementGameCloneCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGameCloneCount(ctx, gameID)
}

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

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func randomHash() string {
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return enc.EncodeToString(randomBytes)
}
