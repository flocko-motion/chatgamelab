package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"time"
	db "webapp-server/db/sqlc"
	"webapp-server/obj"

	"github.com/google/uuid"
)

type GetGamesFilters struct {
	PublicOnly bool
}

// GetGames returns games based on filters. If userID is provided, returns user's games.
// If PublicOnly filter is set, returns only public games.
func GetGames(ctx context.Context, userID *uuid.UUID, filters *GetGamesFilters) ([]obj.Game, error) {
	var dbGames []db.Game
	var err error

	if filters != nil && filters.PublicOnly {
		dbGames, err = queries().GetPublicGames(ctx)
	} else if userID != nil {
		// TODO: we need to add complex selection of games owned by workshops that the user is allowed to see to to an institution membership.. we'll leave that for later.
		dbGames, err = queries().GetGamesVisibleToUser(ctx, uuid.NullUUID{UUID: *userID, Valid: true})
	} else {
		return nil, errors.New("must provide userID or set PublicOnly filter")
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
		Description:              sql.NullString{String: ptrToString(game.Description), Valid: game.Description != nil},
		Icon:                     game.Icon,
		Public:                   game.Public,
		PublicSponsoredApiKeyID:  uuidPtrToNullUUID(game.PublicSponsoredApiKeyID),
		PrivateShareHash:         sql.NullString{String: ptrToString(game.PrivateShareHash), Valid: game.PrivateShareHash != nil},
		PrivateSponsoredApiKeyID: uuidPtrToNullUUID(game.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    game.SystemMessageScenario,
		SystemMessageGameStart:   game.SystemMessageGameStart,
		ImageStyle:               game.ImageStyle,
		Css:                      sql.NullString{String: ptrToString(game.CSS), Valid: game.CSS != nil},
		StatusFields:             game.StatusFields,
		FirstMessage:             sql.NullString{String: ptrToString(game.FirstMessage), Valid: game.FirstMessage != nil},
		FirstStatus:              sql.NullString{String: ptrToString(game.FirstStatus), Valid: game.FirstStatus != nil},
		FirstImage:               game.FirstImage,
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
		Description:              sql.NullString{String: ptrToString(game.Description), Valid: game.Description != nil},
		Icon:                     game.Icon,
		Public:                   game.Public,
		PublicSponsoredApiKeyID:  uuidPtrToNullUUID(game.PublicSponsoredApiKeyID),
		PrivateShareHash:         privateShareHash,
		PrivateSponsoredApiKeyID: uuidPtrToNullUUID(game.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    game.SystemMessageScenario,
		SystemMessageGameStart:   game.SystemMessageGameStart,
		ImageStyle:               game.ImageStyle,
		Css:                      sql.NullString{String: ptrToString(game.CSS), Valid: game.CSS != nil},
		StatusFields:             game.StatusFields,
		FirstMessage:             sql.NullString{String: ptrToString(game.FirstMessage), Valid: game.FirstMessage != nil},
		FirstStatus:              sql.NullString{String: ptrToString(game.FirstStatus), Valid: game.FirstStatus != nil},
		FirstImage:               game.FirstImage,
	}

	_, err = queries().UpdateGame(ctx, arg)
	return err
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

	return &obj.Game{
		ID: g.ID,
		Meta: obj.Meta{
			CreatedBy:  g.CreatedBy,
			CreatedAt:  &g.CreatedAt,
			ModifiedBy: g.ModifiedBy,
			ModifiedAt: &g.ModifiedAt,
		},
		Name:                     g.Name,
		Description:              nullStringToPtr(g.Description),
		Icon:                     g.Icon,
		Public:                   g.Public,
		PublicSponsoredApiKeyID:  nullUUIDToPtr(g.PublicSponsoredApiKeyID),
		PrivateShareHash:         nullStringToPtr(g.PrivateShareHash),
		PrivateSponsoredApiKeyID: nullUUIDToPtr(g.PrivateSponsoredApiKeyID),
		SystemMessageScenario:    g.SystemMessageScenario,
		SystemMessageGameStart:   g.SystemMessageGameStart,
		ImageStyle:               g.ImageStyle,
		CSS:                      nullStringToPtr(g.Css),
		StatusFields:             g.StatusFields,
		FirstMessage:             nullStringToPtr(g.FirstMessage),
		FirstStatus:              nullStringToPtr(g.FirstStatus),
		FirstImage:               g.FirstImage,
		Tags:                     tags,
	}, nil
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
