// package: db / database access and repository layer
// type:    data
// job:     conversion helpers between sqlc rows and obj types, plus sql.Null* helpers.
// limits:  holds no queries or business logic (-> game.go and siblings).
package db

import (
	db "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

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
		Name:                         g.Name,
		Description:                  g.Description,
		Icon:                         g.Icon,
		Public:                       g.Public,
		PublicSponsoredApiKeyShareID: nullUUIDToPtr(g.PublicSponsoredApiKeyShareID),
		SystemMessageScenario:        g.SystemMessageScenario,
		SystemMessageGameStart:       g.SystemMessageGameStart,
		ImageStyle:                   g.ImageStyle,
		CSS:                          g.Css,
		StatusFields:                 g.StatusFields,
		Theme:                        theme,
		FirstMessage:                 nullStringToPtr(g.FirstMessage),
		FirstStatus:                  nullStringToPtr(g.FirstStatus),
		FirstImage:                   g.FirstImage,
		Tags:                         tags,
		PlayCount:                    int(g.PlayCount),
		CloneCount:                   int(g.CloneCount),
		OriginallyCreatedBy:          nullUUIDToPtr(g.OriginallyCreatedBy),
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
