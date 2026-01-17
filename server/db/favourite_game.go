package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"fmt"

	"github.com/google/uuid"
)

// AddFavouriteGame adds a game to a user's favourites. If already a favourite, does nothing.
func AddFavouriteGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	arg := db.AddFavouriteGameParams{
		CreatedBy: uuid.NullUUID{UUID: userID, Valid: true},
		GameID:    gameID,
	}
	_, err := queries().AddFavouriteGame(ctx, arg)
	if err != nil {
		// ON CONFLICT DO NOTHING returns no rows, which causes sql.ErrNoRows
		// This is expected behavior when the favourite already exists
		if err.Error() == "sql: no rows in result set" {
			return nil
		}
		return fmt.Errorf("failed to add favourite game: %w", err)
	}
	return nil
}

// RemoveFavouriteGame removes a game from a user's favourites.
func RemoveFavouriteGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) error {
	arg := db.RemoveFavouriteGameParams{
		UserID: userID,
		GameID: gameID,
	}
	return queries().RemoveFavouriteGame(ctx, arg)
}

// GetFavouriteGames returns all favourite games for a user.
func GetFavouriteGames(ctx context.Context, userID uuid.UUID) ([]obj.Game, error) {
	dbGames, err := queries().GetFavouriteGamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favourite games: %w", err)
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

// IsFavouriteGame checks if a game is in a user's favourites.
func IsFavouriteGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (bool, error) {
	arg := db.IsFavouriteGameParams{
		UserID: userID,
		GameID: gameID,
	}
	return queries().IsFavouriteGame(ctx, arg)
}
