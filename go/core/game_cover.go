package core

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/obj"

	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"github.com/google/uuid"
)

// A game's cover is part of its card: whoever may read the card sees it, and
// whoever may edit the card has it made. It is generated, never uploaded,
// so the youth-protection rule binds every picture (session/cover.go).

// GameCover is a game's cover picture and when it was made.
type GameCover struct {
	Image     []byte
	CreatedAt time.Time
}

// ReadGameCover is a game's cover, if the caller may read its card: {Read, GameCard}. A game they may not read, or
// one without a cover, is ErrNotFound.
func ReadGameCover(ctx context.Context, id uuid.UUID) (*GameCover, error) {
	if _, err := ReadGameCard(ctx, id); err != nil {
		return nil, err
	}
	row, err := store.Queries().V2GameCover(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &GameCover{Image: row.Image, CreatedAt: row.CreatedAt}, nil
}

// CoverFacts are what a game's cover is made from: the game's own words and
// the youth-protection rule the picture keeps to.
type CoverFacts struct {
	GameID      uuid.UUID
	Title       string
	Description string
	Scenario    string
	ImageStyle  string
	// Rule is the site's rule for the youngest players: a cover shows on the
	// card, to anyone who may see the game, so it keeps to the strictest.
	Rule *string
}

// ReadCoverFacts reads what the caller needs to have a game's cover made:
// making it is editing the card, {Edit, GameCard}, and it draws on the
// content, {Read, GameContent}. A game they may not read is ErrNotFound.
func ReadCoverFacts(ctx context.Context, id uuid.UUID) (*CoverFacts, error) {
	game, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !game.MayEditCard() {
		return nil, forbidden("you may not change this game's cover")
	}
	content, err := game.AsGameContent()
	if err != nil {
		return nil, err
	}
	settings, err := store.Queries().V2GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &CoverFacts{
		GameID: game.ID(), Title: game.Name(), Description: game.Description(),
		Scenario: content.Scenario(), ImageStyle: content.ImageStyle(),
		Rule: nullString(settings.PromptConstraintU13),
	}, nil
}

// SetGameCover stores a game's new cover, replacing the one it had, and returns the card as it now is: {Edit, GameCard}.
func SetGameCover(ctx context.Context, id uuid.UUID, image []byte) (*GameCard, error) {
	game, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !game.MayEditCard() {
		return nil, forbidden("you may not change this game's cover")
	}
	if len(image) == 0 {
		return nil, refuse("a cover needs a picture")
	}
	err = store.Queries().V2SetGameCover(ctx, db.V2SetGameCoverParams{
		GameID: id, Image: image, CreatedBy: account(ctx).UUID,
	})
	if err != nil {
		return nil, err
	}
	return ReadGameCard(ctx, id)
}

// CoverRefusal turns why a cover could not be made into an answer the
// editor tells in its own words: no key that draws pictures (400,
// no_api_key), a key its provider rejects (400, invalid_api_key), or a
// provider that failed or did not answer (503, ai_error).
func CoverRefusal(err error) error {
	var refusal *engine.Refusal
	if !errors.As(err, &refusal) {
		return schemafapi.Error(schemafapi.ErrUnavailable, obj.ErrCodeAiError, "the cover could not be made: "+err.Error())
	}
	switch refusal.Problem {
	case engine.ProblemNoKey, engine.ProblemGenre:
		return badRequest(obj.ErrCodeNoApiKey, "%s", refusal.Error())
	case engine.ProblemKeyRejected:
		return badRequest(obj.ErrCodeInvalidApiKey, "%s", refusal.Error())
	}
	return schemafapi.Error(schemafapi.ErrUnavailable, obj.ErrCodeAiError, refusal.Error())
}
