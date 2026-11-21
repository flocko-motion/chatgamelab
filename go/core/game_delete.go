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
// Deleting takes the right to edit the game's settings: {Edit, GameSettings}.
// The game stays as a soft-deleted row; what hangs on it goes: tags,
// favorites, sessions, share links with their guests, the game's key shares.
func DeleteGame(ctx context.Context, id uuid.UUID) (*GameCard, error) {
	game, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !game.MayEditSettings() {
		return nil, forbidden("you may not delete this game")
	}
	by := account(ctx).UUID // whoever may edit has an account: a guest owns nothing

	q := store.Queries()
	_ = q.DeleteGameTagsByGameID(ctx, id)
	_ = q.DeleteFavoritesByGameID(ctx, id)
	_, _ = q.DeleteGameSessionsByGameID(ctx, id)
	_ = db.DeleteGuestDataByGameID(ctx, id) // before the share links it hangs on
	_ = q.DeleteGameSharesByGameID(ctx, id)
	_ = q.DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err := q.SoftDeleteGame(ctx, id); err != nil {
		return nil, err
	}

	if w := game.row.workshopID; w != nil {
		events.GetBroker().PublishGameDeleted(*w, id, by)
	}
	return game, nil
}
