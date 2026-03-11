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
	"strings"
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

	// Admins see all games platform-wide for the "all" filter
	if userID != nil && filterType != "own" && filterType != "public" {
		adminUser, _ := GetUserByID(ctx, *userID)
		if adminUser != nil && adminUser.Role != nil && adminUser.Role.Role == obj.RoleAdmin {
			dbGames, err = getAllGames(ctx, searchQuery, sortField, sortDir)
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

// getAllGames fetches all games platform-wide (for admin use) with optional search and sorting
func getAllGames(ctx context.Context, search, sortField, sortDir string) ([]db.Game, error) {
	searchStr := sql.NullString{String: search, Valid: search != ""}

	if search != "" {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				return queries().SearchAllGamesSortedByName(ctx, searchStr)
			}
			return queries().SearchAllGamesSortedByNameDesc(ctx, searchStr)
		case "createdAt":
			if sortDir == "asc" {
				return queries().SearchAllGamesSortedByCreatedAt(ctx, searchStr)
			}
			return queries().SearchAllGames(ctx, searchStr)
		case "modifiedAt":
			if sortDir == "asc" {
				return queries().SearchAllGamesSortedByModifiedAtAsc(ctx, searchStr)
			}
			return queries().SearchAllGamesSortedByModifiedAt(ctx, searchStr)
		default:
			return queries().SearchAllGames(ctx, searchStr)
		}
	}

	switch sortField {
	case "name":
		if sortDir == "asc" {
			return queries().GetAllGamesSortedByName(ctx)
		}
		return queries().GetAllGamesSortedByNameDesc(ctx)
	case "createdAt":
		if sortDir == "asc" {
			return queries().GetAllGamesSortedByCreatedAt(ctx)
		}
		return queries().GetAllGames(ctx)
	case "modifiedAt":
		if sortDir == "asc" {
			return queries().GetAllGamesSortedByModifiedAtAsc(ctx)
		}
		return queries().GetAllGamesSortedByModifiedAt(ctx)
	default:
		return queries().GetAllGames(ctx)
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

	var games []db.Game

	if search != "" {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				games, err = queries().SearchGamesVisibleToUserSortedByName(ctx, db.SearchGamesVisibleToUserSortedByNameParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			} else {
				games, err = queries().SearchGamesVisibleToUserSortedByNameDesc(ctx, db.SearchGamesVisibleToUserSortedByNameDescParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			}
		case "createdAt":
			if sortDir == "asc" {
				games, err = queries().SearchGamesVisibleToUserSortedByCreatedAt(ctx, db.SearchGamesVisibleToUserSortedByCreatedAtParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			} else {
				games, err = queries().SearchGamesVisibleToUser(ctx, db.SearchGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			}
		case "modifiedAt":
			if sortDir == "asc" {
				games, err = queries().SearchGamesVisibleToUserSortedByModifiedAtAsc(ctx, db.SearchGamesVisibleToUserSortedByModifiedAtAscParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			} else {
				games, err = queries().SearchGamesVisibleToUserSortedByModifiedAt(ctx, db.SearchGamesVisibleToUserSortedByModifiedAtParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			}
		default:
			games, err = queries().SearchGamesVisibleToUser(ctx, db.SearchGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
		}
	} else {
		switch sortField {
		case "name":
			if sortDir == "asc" {
				games, err = queries().GetGamesVisibleToUserSortedByName(ctx, db.GetGamesVisibleToUserSortedByNameParams{CreatedBy: userParam, WorkshopID: workshopParam})
			} else {
				games, err = queries().GetGamesVisibleToUserSortedByNameDesc(ctx, db.GetGamesVisibleToUserSortedByNameDescParams{CreatedBy: userParam, WorkshopID: workshopParam})
			}
		case "createdAt":
			if sortDir == "asc" {
				games, err = queries().GetGamesVisibleToUserSortedByCreatedAt(ctx, db.GetGamesVisibleToUserSortedByCreatedAtParams{CreatedBy: userParam, WorkshopID: workshopParam})
			} else {
				games, err = queries().GetGamesVisibleToUser(ctx, db.GetGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam})
			}
		case "modifiedAt":
			if sortDir == "asc" {
				games, err = queries().GetGamesVisibleToUserSortedByModifiedAtAsc(ctx, db.GetGamesVisibleToUserSortedByModifiedAtAscParams{CreatedBy: userParam, WorkshopID: workshopParam})
			} else {
				games, err = queries().GetGamesVisibleToUserSortedByModifiedAt(ctx, db.GetGamesVisibleToUserSortedByModifiedAtParams{CreatedBy: userParam, WorkshopID: workshopParam})
			}
		default:
			games, err = queries().GetGamesVisibleToUser(ctx, db.GetGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam})
		}
	}

	if err != nil {
		return nil, err
	}

	// Apply workshop visibility settings for participants/individuals only.
	// Head/staff always see all workshop games regardless of these settings.
	isHeadOrStaff := user != nil && user.Role != nil &&
		(user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff)
	if user != nil && user.Role != nil && user.Role.Workshop != nil && !isHeadOrStaff {
		ws := user.Role.Workshop
		filtered := make([]db.Game, 0, len(games))
		for _, g := range games {
			isWorkshopGame := g.WorkshopID.Valid && g.WorkshopID.UUID == ws.ID

			// Public games (from anywhere): controlled by showPublicGames
			if g.Public {
				if ws.ShowPublicGames {
					filtered = append(filtered, g)
				}
				continue
			}

			// Non-public games must belong to this workshop
			if !isWorkshopGame {
				continue
			}

			// Own workshop games always visible
			if g.CreatedBy.Valid && g.CreatedBy.UUID == userID {
				filtered = append(filtered, g)
				continue
			}

			// Other people's workshop games: controlled by showOtherParticipantsGames
			if ws.ShowOtherParticipantsGames {
				filtered = append(filtered, g)
			}
		}
		games = filtered
	}

	return games, nil
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

// GetGameByIDWithShareToken loads a game by ID, granting read access via a share token.
// Used by guest play endpoints where the share token proves access to non-public games.
func GetGameByIDWithShareToken(ctx context.Context, gameID uuid.UUID, shareToken string) (*obj.Game, error) {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGame(ctx, uuid.Nil, OpRead, game, &shareToken); err != nil {
		return nil, err
	}
	return game, nil
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
		ID:                            game.ID,
		CreatedBy:                     uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:                     now,
		ModifiedBy:                    uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:                    now,
		Name:                          game.Name,
		Description:                   game.Description,
		Icon:                          game.Icon,
		WorkshopID:                    uuidPtrToNullUUID(game.WorkshopID),
		Public:                        game.Public,
		PublicSponsoredApiKeyShareID:  uuidPtrToNullUUID(game.PublicSponsoredApiKeyShareID),
		SystemMessageScenario:         game.SystemMessageScenario,
		SystemMessageGameStart:        game.SystemMessageGameStart,
		ImageStyle:                    game.ImageStyle,
		Css:                           game.CSS,
		StatusFields:                  game.StatusFields,
		Theme:                         themeJSON,
		FirstMessage:                  sql.NullString{String: functional.Deref(game.FirstMessage, ""), Valid: game.FirstMessage != nil},
		FirstStatus:                   sql.NullString{String: functional.Deref(game.FirstStatus, ""), Valid: game.FirstStatus != nil},
		FirstImage:                    game.FirstImage,
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
		Plot:          sql.NullString{String: functional.Deref(msg.Plot, ""), Valid: msg.Plot != nil},
		ImagePrompt:   sql.NullString{String: functional.Deref(msg.ImagePrompt, ""), Valid: msg.ImagePrompt != nil},
		Image:         msg.Image,
		HasImage:      msg.HasImage,
		HasAudio:      msg.HasAudioOut,
		ApiKeyType:    sql.NullString{String: msg.ApiKeyType, Valid: msg.ApiKeyType != ""},
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
		ID:                    msg.ID,
		CreatedBy:             uuid.NullUUID{},
		CreatedAt:             time.Time{},
		ModifiedBy:            uuid.NullUUID{},
		ModifiedAt:            now,
		GameSessionID:         msg.GameSessionID,
		Type:                  msg.Type,
		Message:               msg.Message,
		Status:                statusJSON,
		Plot:                  sql.NullString{String: functional.Deref(msg.Plot, ""), Valid: msg.Plot != nil},
		ImagePrompt:           sql.NullString{String: functional.Deref(msg.ImagePrompt, ""), Valid: msg.ImagePrompt != nil},
		Image:                 msg.Image,
		HasImage:              msg.HasImage,
		HasAudio:              msg.HasAudioOut,
		PromptStatusUpdate:    sql.NullString{String: functional.Deref(msg.PromptStatusUpdate, ""), Valid: msg.PromptStatusUpdate != nil},
		PromptResponseSchema:  sql.NullString{String: functional.Deref(msg.PromptResponseSchema, ""), Valid: msg.PromptResponseSchema != nil},
		PromptImageGeneration: sql.NullString{String: functional.Deref(msg.PromptImageGeneration, ""), Valid: msg.PromptImageGeneration != nil},
		PromptExpandStory:     sql.NullString{String: functional.Deref(msg.PromptExpandStory, ""), Valid: msg.PromptExpandStory != nil},
		ResponseRaw:           sql.NullString{String: functional.Deref(msg.ResponseRaw, ""), Valid: msg.ResponseRaw != nil},
		TokenUsage:            tokenUsageJSON,
		UrlAnalytics:          sql.NullString{String: functional.Deref(msg.URLAnalytics, ""), Valid: msg.URLAnalytics != nil},
		ApiKeyType:            sql.NullString{String: msg.ApiKeyType, Valid: msg.ApiKeyType != ""},
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
		ID:         s.ID,
		GameID:     s.GameID,
		UserID:     s.UserID,
		ApiKeyID:   nullUUIDToPtr(s.ApiKeyID),
		AiPlatform: s.AiPlatform,
		AiModel:    s.AiModel,
		AiSession:  string(s.AiSession),
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

	// Load workshop prompt constraints from user's role (if in a workshop)
	user, err := GetUserByID(ctx, s.UserID)
	if err == nil && user.Role != nil && user.Role.Workshop != nil && user.Role.Workshop.PromptConstraints != nil {
		constraints := strings.TrimSpace(*user.Role.Workshop.PromptConstraints)
		if constraints != "" {
			session.WorkshopPromptConstraints = &constraints
		}
	}

	return session, nil
}

// ResolveAndUpdateGameSessionApiKey re-resolves the API key for a session using the standard
// priority chain (workshop → sponsored game → institution free-use → user default → system free-use) and updates the session.
// Used when resuming a session whose API key was deleted.
// Only accepts keys from the same AI platform as the session to prevent mid-session platform switches.
func ResolveAndUpdateGameSessionApiKey(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*obj.GameSession, error) {
	// Load and verify session ownership
	session, err := loadSessionByID(ctx, sessionID)
	if err != nil {
		return nil, obj.ErrNotFound("session not found")
	}
	if session.UserID != userID {
		return nil, obj.ErrForbidden("not the owner of this session")
	}

	// Only accept keys from the same platform as the session.
	// Switching platforms mid-session would break AiSession state (platform-specific conversation IDs).
	requiredPlatform := session.AiPlatform

	// matchesPlatform returns false if the share's platform doesn't match the session's locked platform.
	matchesPlatform := func(s *obj.ApiKeyShare) bool {
		return s != nil && s.ApiKey != nil && (requiredPlatform == "" || s.ApiKey.Platform == requiredPlatform)
	}

	// Resolve the API key and AI quality tier using the same priority chain as session creation:
	// 1. Workshop key + tier → 2. Sponsored game key → 3. Institution free-use key + tier → 4. User default key + tier → 5. System free-use key + tier
	var share *obj.ApiKeyShare
	var sourceTier *string

	user, userErr := GetUserByID(ctx, userID)

	// Load system settings for default tier fallback
	settings, _ := GetSystemSettings(ctx)
	defaultTier := obj.AiModelBalanced
	if settings != nil && settings.DefaultAiQualityTier != "" {
		defaultTier = settings.DefaultAiQualityTier
	}

	// 1. Check for workshop key
	if userErr == nil && user.Role != nil && user.Role.Workshop != nil {
		workshop, wsErr := GetWorkshopByID(ctx, userID, user.Role.Workshop.ID)
		if wsErr == nil && workshop.DefaultApiKeyShareID != nil {
			candidate, _ := GetApiKeyShareByID(ctx, userID, *workshop.DefaultApiKeyShareID)
			if matchesPlatform(candidate) {
				share = candidate
				sourceTier = workshop.AiQualityTier
			}
		}
	}

	// 2. Check sponsored game key
	if share == nil {
		game, gameErr := loadGameByID(ctx, session.GameID)
		if gameErr == nil && game.PublicSponsoredApiKeyShareID != nil {
			candidate, shareErr := GetApiKeyShareByID(ctx, userID, *game.PublicSponsoredApiKeyShareID)
			if shareErr == nil && matchesPlatform(candidate) {
				share = candidate
			}
		}
	}

	// 3. Check institution free-use key
	if share == nil && userErr == nil && user.Role != nil && user.Role.Institution != nil && user.Role.Institution.FreeUseApiKeyShareID != nil {
		candidate, _ := GetApiKeyShareByID(ctx, userID, *user.Role.Institution.FreeUseApiKeyShareID)
		if matchesPlatform(candidate) {
			share = candidate
			institution, instErr := GetInstitutionByID(ctx, userID, user.Role.Institution.ID)
			if instErr == nil {
				sourceTier = institution.FreeUseAiQualityTier
			}
		}
	}

	// 4. Check user's default API key (is_default=true on api_key table)
	if share == nil && userErr == nil {
		defaultKey, _ := GetDefaultApiKeyForUser(ctx, userID)
		if defaultKey != nil {
			candidate, _ := GetSelfShareForApiKey(ctx, userID, defaultKey.ID)
			if matchesPlatform(candidate) {
				share = candidate
				sourceTier = user.AiQualityTier
			}
		}
	}

	// 5. Check system free-use key (stored as api_key_id, not a share)
	if share == nil && settings != nil && settings.FreeUseApiKeyID != nil {
		apiKey, keyErr := GetApiKeyByID(ctx, *settings.FreeUseApiKeyID)
		if keyErr == nil {
			candidate := &obj.ApiKeyShare{
				ApiKeyID: apiKey.ID,
				ApiKey:   apiKey,
			}
			if matchesPlatform(candidate) {
				share = candidate
				sourceTier = settings.FreeUseAiQualityTier
			}
		}
	}

	if share == nil || share.ApiKey == nil {
		if requiredPlatform != "" {
			return nil, &obj.HTTPError{StatusCode: 400, Code: obj.ErrCodeNoApiKey, Message: "No API key available for platform " + requiredPlatform + ". All available keys use a different AI platform."}
		}
		return nil, &obj.HTTPError{StatusCode: 400, Code: obj.ErrCodeNoApiKey, Message: "No API key available. Please configure an API key in your settings."}
	}

	// Resolve the AI model tier: source tier → system default → hardcoded fallback
	aiModel := defaultTier
	if sourceTier != nil && *sourceTier != "" {
		aiModel = *sourceTier
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

// SetGamePublicSponsorship sets a public sponsorship on a game.
// Creates a game-scoped API key share and links it to the game.
// The user must own the API key and have permission to update the game.
func SetGamePublicSponsorship(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, apiKeyShareID uuid.UUID) error {
	// Load game and check update permission
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if err := canAccessGame(ctx, userID, OpUpdate, game, nil); err != nil {
		return err
	}

	// Verify the share exists and the user is authorized to use it
	share, err := queries().GetApiKeyShareByID(ctx, apiKeyShareID)
	if err != nil {
		return obj.ErrNotFound("api key share not found")
	}
	if err := canUseShareForSponsoring(ctx, userID, share); err != nil {
		return err
	}

	// Verify the key has been proven to work (last_usage_success must be true)
	if share.KeyLastUsageSuccess.Valid && !share.KeyLastUsageSuccess.Bool {
		return obj.ErrValidation("api key must be proven to work before sponsoring")
	}

	// Remove any existing sponsorship and its game-scoped shares first
	if game.PublicSponsoredApiKeyShareID != nil {
		if err := queries().ClearGamePublicSponsor(ctx, gameID); err != nil {
			return obj.ErrServerError("failed to clear existing sponsorship")
		}
		if err := queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true}); err != nil {
			log.Debug("failed to delete old game-scoped shares", "game_id", gameID, "error", err)
		}
	}

	// Create a game-scoped share for this sponsorship
	sponsorShareID, err := createApiKeyShareInternal(ctx, userID, share.ApiKeyID, &userID, nil, nil, &gameID)
	if err != nil {
		return obj.ErrServerError("failed to create sponsorship share")
	}

	// Set the sponsor share on the game
	if err := queries().SetGamePublicSponsor(ctx, db.SetGamePublicSponsorParams{
		ID:                           gameID,
		PublicSponsoredApiKeyShareID: uuid.NullUUID{UUID: *sponsorShareID, Valid: true},
	}); err != nil {
		return obj.ErrServerError("failed to set game sponsorship")
	}

	return nil
}

// ClearGamePublicSponsorship removes the public sponsorship from a game.
// Also deletes the game-scoped API key share.
// Allowed by: game owner (OpUpdate) OR the API key owner who sponsors the game.
func ClearGamePublicSponsorship(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return err
	}

	if game.PublicSponsoredApiKeyShareID == nil {
		return nil // Already no sponsorship
	}

	// Allow if user can update the game (game owner / higher role)
	accessErr := canAccessGame(ctx, userID, OpUpdate, game, nil)
	if accessErr != nil {
		// Also allow if user owns the API key behind the sponsorship
		share, shareErr := queries().GetApiKeyShareByID(ctx, *game.PublicSponsoredApiKeyShareID)
		if shareErr != nil {
			return accessErr
		}
		key, keyErr := queries().GetApiKeyByID(ctx, share.ApiKeyID)
		if keyErr != nil || key.UserID != userID {
			return accessErr
		}
	}

	// Clear the sponsor on the game
	if err := queries().ClearGamePublicSponsor(ctx, gameID); err != nil {
		return obj.ErrServerError("failed to clear game sponsorship")
	}

	// Delete game-scoped shares for this game
	if err := queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true}); err != nil {
		log.Debug("failed to delete game-scoped shares", "game_id", gameID, "error", err)
	}

	return nil
}

// ClearGamePublicSponsorshipByShareID removes sponsorship if the given share is the sponsor.
// Used when auto-removing sponsorship on key failure.
func ClearGamePublicSponsorshipByShareID(ctx context.Context, gameID uuid.UUID, shareID uuid.UUID) error {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game.PublicSponsoredApiKeyShareID == nil || *game.PublicSponsoredApiKeyShareID != shareID {
		return nil // Not the sponsor
	}

	if err := queries().ClearGamePublicSponsor(ctx, gameID); err != nil {
		return err
	}

	// Delete game-scoped shares for this game
	if err := queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true}); err != nil {
		log.Debug("failed to delete game-scoped shares", "game_id", gameID, "error", err)
	}

	return nil
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

	// Deserialize theme from JSON if present
	var theme *obj.GameTheme
	if g.Theme.Valid && len(g.Theme.RawMessage) > 0 {
		theme = &obj.GameTheme{}
		if err := json.Unmarshal(g.Theme.RawMessage, theme); err != nil {
			log.Warn("failed to unmarshal game theme", "game_id", g.ID, "error", err)
			theme = nil
		}
	}

	game := &obj.Game{
		ID: g.ID,
		Meta: obj.Meta{
			CreatedBy:  g.CreatedBy,
			CreatedAt:  &g.CreatedAt,
			ModifiedBy: g.ModifiedBy,
			ModifiedAt: &g.ModifiedAt,
		},
		Name:                          g.Name,
		Description:                   g.Description,
		Icon:                          g.Icon,
		Public:                        g.Public,
		PublicSponsoredApiKeyShareID:  nullUUIDToPtr(g.PublicSponsoredApiKeyShareID),
		SystemMessageScenario:         g.SystemMessageScenario,
		SystemMessageGameStart:        g.SystemMessageGameStart,
		ImageStyle:                    g.ImageStyle,
		CSS:                           g.Css,
		StatusFields:                  g.StatusFields,
		Theme:                         theme,
		FirstMessage:                  nullStringToPtr(g.FirstMessage),
		FirstStatus:                   nullStringToPtr(g.FirstStatus),
		FirstImage:                    g.FirstImage,
		Tags:                          tags,
		PlayCount:                     int(g.PlayCount),
		CloneCount:                    int(g.CloneCount),
		OriginallyCreatedBy:           nullUUIDToPtr(g.OriginallyCreatedBy),
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

// IncrementGamePlayCount increments the play count of a game by 1.
func IncrementGamePlayCount(ctx context.Context, gameID uuid.UUID) error {
	return queries().IncrementGamePlayCount(ctx, gameID)
}

// CreateGameShare creates a game share link with a game-scoped API key share.
// The sourceShareID is the user's personal/workshop share that will be cloned into a game-scoped share.
func CreateGameShare(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, sourceShareID uuid.UUID, institutionID, workshopID *uuid.UUID, maxSessions *int) (*obj.GameShare, error) {
	// Verify the source share exists and the user is authorized to use it
	share, err := queries().GetApiKeyShareByID(ctx, sourceShareID)
	if err != nil {
		return nil, obj.ErrNotFound("api key share not found")
	}
	// For workshop shares, the route handler already verified workshop access and sharing permissions.
	// Only check personal sponsoring permissions for non-workshop shares.
	if workshopID == nil {
		if err := canUseShareForSponsoring(ctx, userID, share); err != nil {
			return nil, err
		}
	}

	// Verify the key hasn't been proven to NOT work
	if share.KeyLastUsageSuccess.Valid && !share.KeyLastUsageSuccess.Bool {
		return nil, obj.ErrValidation("api key must be working before it can be used for sharing")
	}

	// Create a game-scoped API key share (accessible by uuid.Nil in guest play flow)
	gameScopedShareID, err := createApiKeyShareInternal(ctx, userID, share.ApiKeyID, &userID, nil, nil, &gameID)
	if err != nil {
		return nil, obj.ErrServerError("failed to create game-scoped share")
	}

	// Generate a secure token for the share link
	token, err := functional.GenerateSecureToken(20)
	if err != nil {
		return nil, obj.ErrServerError("failed to generate share token")
	}

	// Create the game_share row
	gs, err := queries().CreateGameShare(ctx, db.CreateGameShareParams{
		GameID:        gameID,
		Token:         token,
		ApiKeyShareID: *gameScopedShareID,
		InstitutionID: uuidPtrToNullUUID(institutionID),
		WorkshopID:    uuidPtrToNullUUID(workshopID),
		Remaining:     intPtrToNullInt32(maxSessions),
		CreatedBy:     uuid.NullUUID{UUID: userID, Valid: true},
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to create game share")
	}

	return dbGameShareToObj(gs), nil
}

// GetGameShareByToken loads a game share by its token.
func GetGameShareByToken(ctx context.Context, token string) (*obj.GameShare, error) {
	gs, err := queries().GetGameShareByToken(ctx, token)
	if err != nil {
		return nil, obj.ErrNotFound("share not found")
	}
	return dbGameShareToObj(gs), nil
}

// GetGameShareByID loads a game share by its ID.
func GetGameShareByID(ctx context.Context, shareID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().GetGameShareByID(ctx, shareID)
	if err != nil {
		return nil, obj.ErrNotFound("share not found")
	}
	return dbGameShareToObj(gs), nil
}

// GetGameSharesByGameIDAndCreator returns shares for a game created by a specific user.
func GetGameSharesByGameIDAndCreator(ctx context.Context, gameID uuid.UUID, userID uuid.UUID) ([]obj.GameShare, error) {
	rows, err := queries().GetGameSharesByGameIDAndCreator(ctx, db.GetGameSharesByGameIDAndCreatorParams{
		GameID:    gameID,
		CreatedBy: uuid.NullUUID{UUID: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]obj.GameShare, len(rows))
	for i, r := range rows {
		result[i] = *dbGameShareToObj(r)
	}
	return result, nil
}

// GetGameSharesByGameIDAndWorkshop returns shares for a game in a specific workshop.
func GetGameSharesByGameIDAndWorkshop(ctx context.Context, gameID uuid.UUID, workshopID uuid.UUID) ([]obj.GameShare, error) {
	rows, err := queries().GetGameSharesByGameIDAndWorkshop(ctx, db.GetGameSharesByGameIDAndWorkshopParams{
		GameID:     gameID,
		WorkshopID: uuid.NullUUID{UUID: workshopID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]obj.GameShare, len(rows))
	for i, r := range rows {
		result[i] = *dbGameShareToObj(r)
	}
	return result, nil
}

// GetGameSharesByGameIDAndInstitution returns org-level shares (non-workshop) for a game.
func GetGameSharesByGameIDAndInstitution(ctx context.Context, gameID uuid.UUID, institutionID uuid.UUID) ([]obj.GameShare, error) {
	rows, err := queries().GetGameSharesByGameIDAndInstitution(ctx, db.GetGameSharesByGameIDAndInstitutionParams{
		GameID:        gameID,
		InstitutionID: uuid.NullUUID{UUID: institutionID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]obj.GameShare, len(rows))
	for i, r := range rows {
		result[i] = *dbGameShareToObj(r)
	}
	return result, nil
}

// GetWorkshopGameShare finds an existing workshop share for a game (for reuse).
func GetWorkshopGameShare(ctx context.Context, gameID uuid.UUID, workshopID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().GetWorkshopGameShare(ctx, db.GetWorkshopGameShareParams{
		GameID:     gameID,
		WorkshopID: uuid.NullUUID{UUID: workshopID, Valid: true},
	})
	if err != nil {
		return nil, obj.ErrNotFound("workshop share not found")
	}
	return dbGameShareToObj(gs), nil
}

// GameShareWithGame represents a game share enriched with game name.
type GameShareWithGame struct {
	obj.GameShare
	GameName string
}

// GetGameSharesWithGameByApiKeyID returns game shares (with game name) for all shares of an API key.
func GetGameSharesWithGameByApiKeyID(ctx context.Context, apiKeyID uuid.UUID) ([]GameShareWithGame, error) {
	rows, err := queries().GetGameSharesWithGameByApiKeyID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}
	result := make([]GameShareWithGame, len(rows))
	for i, r := range rows {
		result[i] = GameShareWithGame{
			GameShare: *dbGameShareToObjFromJoin(r.ID, r.GameID, r.Token, r.ApiKeyShareID, r.InstitutionID, r.WorkshopID, r.Remaining, r.CreatedBy, r.CreatedAt),
			GameName:  r.GameName,
		}
	}
	return result, nil
}

// dbGameShareToObjFromJoin converts individual fields (from JOIN query results) to obj.GameShare.
func dbGameShareToObjFromJoin(id, gameID uuid.UUID, token string, apiKeyShareID uuid.UUID, institutionID, workshopID uuid.NullUUID, remaining sql.NullInt32, createdBy uuid.NullUUID, createdAt time.Time) *obj.GameShare {
	gs := &obj.GameShare{
		ID:            id,
		GameID:        gameID,
		Token:         token,
		ApiKeyShareID: apiKeyShareID,
		CreatedAt:     createdAt,
	}
	if institutionID.Valid {
		gs.InstitutionID = &institutionID.UUID
	}
	if workshopID.Valid {
		gs.WorkshopID = &workshopID.UUID
	}
	if remaining.Valid {
		r := int(remaining.Int32)
		gs.Remaining = &r
	}
	if createdBy.Valid {
		gs.CreatedBy = &createdBy.UUID
	}
	return gs
}

// DeleteGameShare deletes a game share and cleans up associated guest data and the game-scoped API key share.
func DeleteGameShare(ctx context.Context, shareID uuid.UUID) error {
	gs, err := queries().GetGameShareByID(ctx, shareID)
	if err != nil {
		return obj.ErrNotFound("share not found")
	}

	// Clean up guest data linked to this share
	_ = DeleteGuestDataByShareID(ctx, shareID)

	// Delete the game_share row
	if err := queries().DeleteGameShare(ctx, shareID); err != nil {
		return err
	}

	// Delete the game-scoped API key share
	_ = queries().DeleteApiKeyShare(ctx, gs.ApiKeyShareID)

	return nil
}

// UpdateGameShareRemaining updates the remaining sessions on a game share. Pass nil for unlimited.
func UpdateGameShareRemaining(ctx context.Context, shareID uuid.UUID, remaining *int) (*obj.GameShare, error) {
	var nullRemaining sql.NullInt32
	if remaining != nil {
		nullRemaining = sql.NullInt32{Int32: int32(*remaining), Valid: true}
	}
	gs, err := queries().UpdateGameShareRemaining(ctx, db.UpdateGameShareRemainingParams{
		ID:        shareID,
		Remaining: nullRemaining,
	})
	if err != nil {
		return nil, err
	}
	return dbGameShareToObj(gs), nil
}

// DecrementGameShareRemaining atomically decrements the remaining counter on a game share.
func DecrementGameShareRemaining(ctx context.Context, shareID uuid.UUID) (*obj.GameShare, error) {
	gs, err := queries().DecrementGameShareRemaining(ctx, shareID)
	if err != nil {
		return nil, obj.ErrForbidden("share link has reached its play limit")
	}
	return dbGameShareToObj(gs), nil
}

// CreateGuestUser creates an anonymous user for guest play sessions.
// shareID links the guest to the game_share for cleanup on revoke.
func CreateGuestUser(ctx context.Context, userID uuid.UUID, name string, shareID uuid.UUID) error {
	_, err := queries().CreateGuestUser(ctx, db.CreateGuestUserParams{
		ID:             userID,
		Name:           name,
		PrivateShareID: uuid.NullUUID{UUID: shareID, Valid: true},
	})
	return err
}

// DeleteGuestDataByShareID removes all guest users, their sessions, and messages
// that were created via a game share link.
// Must delete in order: messages → sessions → users (FK constraints).
func DeleteGuestDataByShareID(ctx context.Context, shareID uuid.UUID) error {
	if err := queries().DeleteGuestSessionMessagesByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true}); err != nil {
		return err
	}
	if err := queries().DeleteGuestSessionsByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true}); err != nil {
		return err
	}
	return queries().DeleteGuestUsersByShareID(ctx, uuid.NullUUID{UUID: shareID, Valid: true})
}

// DeleteGuestDataByGameID removes all guest data for all shares of a game.
func DeleteGuestDataByGameID(ctx context.Context, gameID uuid.UUID) error {
	shares, err := queries().GetGameSharesByGameID(ctx, gameID)
	if err != nil {
		return nil // no shares = nothing to clean up
	}
	for _, gs := range shares {
		_ = DeleteGuestDataByShareID(ctx, gs.ID)
	}
	return nil
}

// dbGameShareToObj converts a DB game_share row to an obj.GameShare.
func dbGameShareToObj(gs db.GameShare) *obj.GameShare {
	result := &obj.GameShare{
		ID:            gs.ID,
		GameID:        gs.GameID,
		Token:         gs.Token,
		ApiKeyShareID: gs.ApiKeyShareID,
		Remaining:     nullInt32ToIntPtr(gs.Remaining),
		CreatedAt:     gs.CreatedAt,
	}
	if gs.InstitutionID.Valid {
		result.InstitutionID = &gs.InstitutionID.UUID
	}
	if gs.WorkshopID.Valid {
		result.WorkshopID = &gs.WorkshopID.UUID
	}
	if gs.CreatedBy.Valid {
		result.CreatedBy = &gs.CreatedBy.UUID
	}
	return result
}

// CountGameSessionMessages returns the number of messages in a game session.
func CountGameSessionMessages(ctx context.Context, sessionID uuid.UUID) (int, error) {
	count, err := queries().CountGameSessionMessages(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func stringPtrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullInt32ToIntPtr(ni sql.NullInt32) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int32)
	return &v
}

func intPtrToNullInt32(i *int) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*i), Valid: true}
}
