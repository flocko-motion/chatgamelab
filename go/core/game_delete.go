package core

import (
	"context"

	"cgl/db"

	"github.com/google/uuid"
)

// DeleteGame deletes a game and returns it as it was.
//
// A game the caller may not read does not exist for them (ErrNotFound).
// Deleting takes the right to edit the game's settings: {Edit, GameSettings}.
func DeleteGame(ctx context.Context, id uuid.UUID) (*GameCard, error) {
	game, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !game.MayEditSettings() {
		return nil, forbidden("you may not delete this game")
	}
	by := account(ctx).UUID // whoever may edit has an account: a guest owns nothing
	if err := db.RemoveGame(ctx, id, game.row.workshopID, by); err != nil {
		return nil, err
	}
	return game, nil
}
