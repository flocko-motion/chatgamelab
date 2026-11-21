package core

import (
	"context"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/events"

	"github.com/google/uuid"
)

// DeleteGame deletes a game and returns it as it was.
//
// A game the caller may not read does not exist for them (ErrNotFound).
// Deleting takes {Delete, GameCard}.
// The game stays as a soft-deleted row; what hangs on it goes: tags,
// favorites, sessions, share links with their guests, the game's key shares.
func DeleteGame(ctx context.Context, id uuid.UUID) (*GameCard, error) {
	game, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !game.MayDelete() {
		return nil, forbidden("you may not delete this game")
	}
	by := account(ctx).UUID // whoever may edit has an account: a guest owns nothing

	// all of it or none
	err = store.InTx(ctx, func(q *db.Queries) error { return deleteGameRows(ctx, q, id, false) })
	if err != nil {
		return nil, err
	}

	if w := game.row.workshopID; w != nil {
		events.GetBroker().PublishGameDeleted(*w, id, by)
	}
	return game, nil
}
