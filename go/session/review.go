// package: session / engine sessions from the platform's data
// type:    logic
// job:     has a game's form reviewed for the editor (the caller's own payers, the engine's ReviewGame), in the caller's language
// limits:  no HTTP, nothing stored; the form is the editor's, as typed, saved or not
package session

import (
	"context"
	"errors"
	"time"

	"cgl/core"
	"cgl/log"
	"engine"
)

// reviewWait bounds a review: the author waits for it.
const reviewWait = 30 * time.Second

// ReviewGameForm finds the one point most worth improving in a game's form,
// in the caller's language. The caller pays as in the editor: their own
// payers, no game's sponsor (core.OwnPayers). A required field left empty
// is found without a call, and without a key.
func ReviewGameForm(ctx context.Context, form engine.ReviewSpec) (engine.Review, error) {
	if !genreKnown(form.Genre) {
		form.Genre = engine.GenreAdventure
	}
	form.Language = playerLanguage(core.PlayerLanguage(ctx), "")
	// An empty required field needs no key to be found.
	if field := form.Missing(); field != "" {
		return engine.Review{Field: field, Missing: true}, nil
	}
	chosen, err := editorPayer(ctx, func(core.PayerOffer) error { return nil }, "no API key is available to review the game")
	if err != nil {
		return engine.Review{}, aiRefusal("the review", err)
	}
	offer, payer := chosen.Offer, chosen.Payer

	ctx, cancel := context.WithTimeout(ctx, reviewWait)
	defer cancel()
	review, err := engine.ReviewGame(ctx, form, offer)
	log.Info("game reviewed", "genre", form.Genre, "platform", offer.Platform, "offer", offer.ID, "source", payer.Source,
		"field", review.Field, "missing", review.Missing, "cost", review.Usage.TotalCost, "priced", review.Usage.Complete, "error", err)
	if errors.Is(err, engine.ErrKeyRejected) {
		payer.RecordCheck(ctx, false, string(engine.ProblemKeyRejected))
	}
	if err != nil {
		return engine.Review{}, aiRefusal("the review", err)
	}
	return review, nil
}
