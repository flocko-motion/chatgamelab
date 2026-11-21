package core

import (
	"context"

	"cgl/core/internal/store"
	"cgl/db"

	"github.com/google/uuid"
)

// A favorite is the caller's own mark on a game, not a right on it: marking
// takes only that the caller may read the game, unmarking nothing at all.

// MarkFavorite marks a game as one of the caller's favorites. A game the
// caller may not read does not exist for them (ErrNotFound). Marking twice
// changes nothing.
func MarkFavorite(ctx context.Context, id uuid.UUID) error {
	if _, err := ReadGameCard(ctx, id); err != nil {
		return err
	}
	caller := account(ctx)
	if !caller.Valid {
		return forbidden("only an account has favorites")
	}
	return store.Queries().V2MarkFavorite(ctx, db.V2MarkFavoriteParams{Caller: caller.UUID, GameID: id})
}

// UnmarkFavorite removes a game from the caller's favorites. It checks
// nothing and answers the same whether the game was marked, exists or is
// readable: it only deletes the caller's own mark, so it gives nothing away,
// and works after the game has become unreadable.
func UnmarkFavorite(ctx context.Context, id uuid.UUID) error {
	caller := account(ctx)
	if !caller.Valid {
		return nil
	}
	return store.Queries().V2UnmarkFavorite(ctx, db.V2UnmarkFavoriteParams{Caller: caller.UUID, GameID: id})
}
