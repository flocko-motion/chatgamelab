// package: session / engine sessions from the platform's data
// type:    logic
// job:     has a game's cover drawn (the first payer whose key draws pictures, the strictest rule, the engine's MakeCover), or a brought-in picture checked against that rule (CheckCover); stored at cover size.
// limits:  no HTTP; reading the game and storing the cover are the secure core's (core/game_cover.go); drawing is the engine's.
package session

import (
	"context"
	"errors"
	"fmt"

	"cgl/core"
	"cgl/log"
	"cgl/obj"
	"cgl/picture"
	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"

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

// ImportGameCover makes a picture brought in from a game's zip a game's
// cover, and returns the card as it now is. Changing the cover takes what
// drawing one does ({Edit, GameCard}). The picture must carry the signature a
// ChatGameLab server gave it on export (cover_unsigned otherwise, before
// anything is paid); it is scaled to the size a cover is stored at, which
// also proves it is one, and checked against the strictest youth-protection
// rule, as a drawn cover keeps to it. The caller pays for the check as for a
// session of the game. A picture that fails it is refused with
// cover_refused, and the game keeps the cover it had.
func ImportGameCover(ctx context.Context, gameID uuid.UUID, upload []byte, signature string) (*core.GameCard, error) {
	facts, err := core.ReadCoverFacts(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if !picture.Signed(upload, signature) {
		return nil, schemafapi.Error(schemafapi.ErrUnprocessable, obj.ErrCodeCoverUnsigned, "the cover was not exported by ChatGameLab, or was changed since")
	}
	image, err := picture.Scale(upload, picture.CoverWidth)
	if err != nil {
		return nil, schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeNotAPicture, "the cover is no PNG or JPEG picture")
	}
	payers, steps, err := core.Payers(ctx, gameID, "")
	if err != nil {
		return nil, err
	}
	traceKeys(gameID, "", steps)
	offers, refused := core.Offers(payers, func(core.PayerOffer) error { return nil })
	chosen, err := firstWorking(ctx, offers, refused, "no API key is available to check the picture")
	if err != nil {
		return nil, core.CoverRefusal(err)
	}
	offer, payer := chosen.Offer, chosen.Payer

	spec := engine.CoverCheckSpec{Image: image}
	if facts.Rule != nil {
		spec.Guardrail = *facts.Rule
	}
	verdict, err := engine.CheckCover(ctx, spec, offer)
	log.Info("cover checked", "game_id", gameID, "platform", offer.Platform, "offer", offer.ID, "source", payer.Source,
		"complies", verdict.Complies, "reason", verdict.Reason, "cost", verdict.Usage.TotalCost, "priced", verdict.Usage.Complete, "error", err)
	if errors.Is(err, engine.ErrKeyRejected) {
		payer.RecordCheck(ctx, false, string(engine.ProblemKeyRejected))
	}
	if err != nil {
		return nil, core.CoverRefusal(err)
	}
	if !verdict.Complies {
		return nil, schemafapi.Error(schemafapi.ErrUnprocessable, obj.ErrCodeCoverRefused, "the picture does not keep to the youth-protection rule: "+verdict.Reason)
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
