// package: db / database access and repository layer
// type:    data
// job:     listing, filtering, and single-game reads.
// limits:  does not write games (-> game_writes.go), sessions (-> game_sessions.go), or shares (-> game_shares.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// GetGamesFilters holds the filter, search, and sort options for listing games.
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
		case "playCount":
			if sortDir == "asc" {
				return queries().SearchAllGamesSortedByPlayCountAsc(ctx, searchStr)
			}
			return queries().SearchAllGamesSortedByPlayCount(ctx, searchStr)
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
	case "playCount":
		if sortDir == "asc" {
			return queries().GetAllGamesSortedByPlayCountAsc(ctx)
		}
		return queries().GetAllGamesSortedByPlayCount(ctx)
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
		case "playCount":
			if sortDir == "asc" {
				games, err = queries().SearchGamesVisibleToUserSortedByPlayCountAsc(ctx, db.SearchGamesVisibleToUserSortedByPlayCountAscParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
			} else {
				games, err = queries().SearchGamesVisibleToUserSortedByPlayCount(ctx, db.SearchGamesVisibleToUserSortedByPlayCountParams{CreatedBy: userParam, WorkshopID: workshopParam, Column3: searchStr})
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
		case "playCount":
			if sortDir == "asc" {
				games, err = queries().GetGamesVisibleToUserSortedByPlayCountAsc(ctx, db.GetGamesVisibleToUserSortedByPlayCountAscParams{CreatedBy: userParam, WorkshopID: workshopParam})
			} else {
				games, err = queries().GetGamesVisibleToUserSortedByPlayCount(ctx, db.GetGamesVisibleToUserSortedByPlayCountParams{CreatedBy: userParam, WorkshopID: workshopParam})
			}
		default:
			games, err = queries().GetGamesVisibleToUser(ctx, db.GetGamesVisibleToUserParams{CreatedBy: userParam, WorkshopID: workshopParam})
		}
	}

	if err != nil {
		return nil, err
	}

	// Apply workshop visibility settings.
	// showPublicGames: applies to ALL roles (controls non-workshop public games)
	// showOtherParticipantsGames: applies to participants/individuals only (head/staff always see all workshop games)
	isHeadOrStaff := user != nil && user.Role != nil &&
		(user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff)
	if user != nil && user.Role != nil && user.Role.Workshop != nil {
		ws := user.Role.Workshop
		filtered := make([]db.Game, 0, len(games))
		for _, g := range games {
			isWsGame := g.WorkshopID.Valid && g.WorkshopID.UUID == ws.ID

			if isWsGame {
				// Head/staff see all workshop games
				if isHeadOrStaff {
					filtered = append(filtered, g)
					continue
				}
				// Others: own games always visible, rest controlled by setting
				if g.CreatedBy.Valid && g.CreatedBy.UUID == userID {
					filtered = append(filtered, g)
				} else if ws.ShowOtherParticipantsGames {
					filtered = append(filtered, g)
				}
				continue
			}

			// Non-workshop public games: controlled by showPublicGames (all roles)
			if g.Public && ws.ShowPublicGames {
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
