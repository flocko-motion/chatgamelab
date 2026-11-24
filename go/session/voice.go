// package: session / engine sessions from the platform's data
// type:    logic
// job:     has a voice say its sample for the game editor (the caller's own payers, the first whose key speaks live, the engine's SpeakSample), and makes a new voice for a voice style
// limits:  no HTTP; the voices and their samples are kept by the secure core (core/voice.go); making and speaking them is the engine's.
package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cgl/core"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"
	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"github.com/google/uuid"
)

// VoiceSampleRequest is the voice the editor wants to hear: the game's genre,
// its voice style and scenario as the form holds them, and whether to make a
// new voice for the style first.
type VoiceSampleRequest struct {
	Genre      string
	VoiceStyle string
	Scenario   string
	Renew      bool
}

// SpeakVoiceSample has the voice a game of the genre would speak in say a
// neutral line, the same for every game, in the caller's language. The
// caller pays as in the editor: their own payers, no game's sponsor
// (core.OwnPayers), the first whose key speaks live. A voice made for the
// style on that key plays its kept sample without asking the provider; one
// without a sample makes it, and the voice first when there is none. Renew
// replaces the voice for that key and style, for every game with the style.
// An empty voice style hears the built-in voice.
func SpeakVoiceSample(ctx context.Context, req VoiceSampleRequest) (engine.VoiceSample, error) {
	genre := engine.Genre(req.Genre)
	if !genreKnown(genre) {
		return engine.VoiceSample{}, schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeValidation, "unknown genre "+req.Genre)
	}
	chosen, err := editorPayer(ctx, speaksLive, "no API key that speaks live is available")
	if err != nil {
		return engine.VoiceSample{}, aiRefusal("the voice", err)
	}
	offer, payer := chosen.Offer, chosen.Payer

	language := playerLanguage(core.PlayerLanguage(ctx), "")
	sample, err := engine.SpeakSample(ctx, engine.VoiceSampleSpec{
		Genre: genre, VoiceStyle: req.VoiceStyle, Scenario: req.Scenario,
		Line: sampleLine(language, genre), Language: language, Renew: req.Renew,
	}, offer)
	log.Info("voice sample", "genre", genre, "platform", offer.Platform, "offer", offer.ID, "source", payer.Source,
		"stored", sample.Stored, "renew", req.Renew, "cost", sample.Usage.TotalCost, "priced", sample.Usage.Complete, "error", err)
	if errors.Is(err, engine.ErrKeyRejected) {
		payer.RecordCheck(ctx, false, string(engine.ProblemKeyRejected))
	}
	if err != nil {
		return engine.VoiceSample{}, aiRefusal("the voice", err)
	}
	return sample, nil
}

// editorPayer is the first of the caller's own payers (core.OwnPayers, no
// game's sponsor) whose offer fits and whose key works, for something the
// caller does in the editor.
func editorPayer(ctx context.Context, fits func(core.PayerOffer) error, none string) (core.PayerOffer, error) {
	payers, steps, err := core.OwnPayers(ctx)
	if err != nil {
		return core.PayerOffer{}, err
	}
	traceKeys(uuid.Nil, "", steps)
	offers, refused := core.Offers(payers, fits)
	return firstWorking(ctx, offers, refused, none)
}

// sampleLine is the line a voice says as its sample: neutral, the same for
// every game, a greeting for a genre whose character talks with the player,
// an opening narration otherwise.
func sampleLine(language string, genre engine.Genre) string {
	key := "common.games.editor.sound.sampleLine.narration"
	if genre == engine.GenreNPCLive {
		key = "common.games.editor.sound.sampleLine.greeting"
	}
	return lang.Text(language, key, nil)
}

func genreKnown(g engine.Genre) bool {
	for _, known := range engine.Genres() {
		if known == g {
			return true
		}
	}
	return false
}

// speaksLive lets through an offer that speaks live at some tier.
func speaksLive(o core.PayerOffer) error {
	if _, ok := engine.LiveTier(o.Offer); !ok {
		return engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("the key on %s speaks no live voice", o.Offer.Platform))
	}
	return nil
}

// aiRefusal is an AI call the editor asked for, refused or failed, as the
// API answers it: no key (400 no_api_key), a key the provider rejects (400
// invalid_api_key), anything else (503 ai_error).
func aiRefusal(what string, err error) error {
	var refusal *engine.Refusal
	if !errors.As(err, &refusal) {
		return schemafapi.Error(schemafapi.ErrUnavailable, obj.ErrCodeAiError, what+" could not be made: "+err.Error())
	}
	switch refusal.Problem {
	case engine.ProblemNoKey, engine.ProblemGenre:
		return schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeNoApiKey, strings.TrimSpace(refusal.Error()))
	case engine.ProblemKeyRejected:
		return schemafapi.Error(schemafapi.ErrBadRequest, obj.ErrCodeInvalidApiKey, refusal.Error())
	}
	return schemafapi.Error(schemafapi.ErrUnavailable, obj.ErrCodeAiError, refusal.Error())
}
