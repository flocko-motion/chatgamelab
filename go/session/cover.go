// package: session / engine sessions from the platform's data
// type:    logic
// job:     has a game's cover drawn: the first payer whose key draws pictures, the strictest rule, the engine's MakeCover, stored at two sizes.
// limits:  no HTTP; reading the game and storing the cover are the secure core's (core/game_cover.go); drawing is the engine's.
package session

import (
	"context"
	"errors"
	"fmt"

	"cgl/core"
	"cgl/log"
	"cgl/picture"
	"engine"

	"github.com/google/uuid"
)

// MakeGameCover draws a game's cover for the caller and stores it, replacing
// the one it had, and returns the game's card as it now is. The caller pays
// as for a session of the game: the first of their payers whose key draws
// pictures (core/payer.go). hint is what the designer wants the cover to
// show; empty leaves it to the game. It blocks until the picture is made.
func MakeGameCover(ctx context.Context, gameID uuid.UUID, hint string) (*core.GameCard, error) {
	facts, err := core.ReadCoverFacts(ctx, gameID)
	if err != nil {
		return nil, err
	}
	payers, steps, err := core.Payers(ctx, gameID, "")
	if err != nil {
		return nil, err
	}
	traceKeys(gameID, "", steps)
	offers, refused := core.Offers(payers, drawsPictures)
	chosen, err := firstWorking(ctx, offers, refused, "no API key that draws pictures is available")
	if err != nil {
		return nil, core.CoverRefusal(err)
	}
	offer, payer := chosen.Offer, chosen.Payer

	spec := engine.CoverSpec{
		Title: facts.Title, Description: facts.Description, Scenario: facts.Scenario,
		ImageStyle: facts.ImageStyle, Hint: hint,
	}
	if facts.Rule != nil {
		spec.Guardrail = *facts.Rule
	}
	cover, err := engine.MakeCover(ctx, spec, offer)
	log.Info("cover drawn", "game_id", gameID, "platform", offer.Platform, "offer", offer.ID, "source", payer.Source,
		"cost", cover.Usage.TotalCost, "priced", cover.Usage.Complete, "error", err)
	if errors.Is(err, engine.ErrKeyRejected) {
		payer.RecordCheck(ctx, false, string(engine.ProblemKeyRejected))
	}
	if err != nil {
		return nil, core.CoverRefusal(err)
	}
	// Stored at the sizes it is shown at; the provider's picture is far larger.
	image, err := picture.Scale(cover.Image, picture.CoverWidth)
	if err != nil {
		return nil, core.CoverRefusal(err)
	}
	return core.SetGameCover(ctx, gameID, image)
}

// drawsPictures lets through an offer that allows pictures at some tier.
func drawsPictures(o core.PayerOffer) error {
	if _, ok := engine.CoverTier(o.Offer); !ok {
		return engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("the key on %s draws no pictures", o.Offer.Platform))
	}
	return nil
}
