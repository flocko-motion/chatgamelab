// package: api / schemaf endpoints
// type:    logic
// job:     lets the game editor hear a game's voice, and make a new one for its voice style
// limits:  keeping voices and samples is the core's (core/voice.go); making and speaking them is session/voice.go's and the engine's
package api

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"unicode/utf8"

	"cgl/session"
	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// maxVoiceText caps the voice style and the scenario sent for a sample, in
// characters: the game's own fields are no longer.
const maxVoiceText = 20000

// VoiceSampleEndpoint has the voice made for a game's voice style say a
// neutral sample line, the same for every game, in the caller's language,
// and answers it as audio. It reads the form as it is, so a game not yet
// saved is heard too.
//
// Only a voice set to custom, in a genre that offers it, is heard (a choice
// in voiceSlots of /api/genres): with a built-in choice the session casts its
// own voice, so a sample would mislead, and anything else is refused with
// 400 voice_not_custom. The voice made for the voice style on the paying key
// speaks: the first sample makes the voice when there is none and keeps the
// sample, and later ones play it without asking the provider (stored: true).
// With an empty voice style a built-in voice speaks.
//
// The caller pays with their own keys, as in the editor: workshop,
// organisation, own, site; never a game's sponsor. None that speaks live:
// 400 no_api_key; a key its provider rejects: 400 invalid_api_key; a
// provider that fails: 503 ai_error.
type VoiceSampleEndpoint struct{}

func (VoiceSampleEndpoint) Method() string             { return "POST" }
func (VoiceSampleEndpoint) Path() string               { return "/api/voices/sample" }
func (VoiceSampleEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (VoiceSampleEndpoint) Handle(ctx context.Context, req VoiceSampleReq) (VoiceSampleResp, error) {
	return speakSample(ctx, req, false)
}

// NewVoiceEndpoint makes a new voice for a voice style on the paying key and
// answers its sample, as VoiceSampleEndpoint does. The voice made before for
// that key and style, and its sample, are deleted: every game with exactly
// this voice style on this key speaks in the new voice from now on. Only for
// a voice set to custom, as VoiceSampleEndpoint; an empty voice style makes
// none and only plays the built-in voice.
type NewVoiceEndpoint struct{}

func (NewVoiceEndpoint) Method() string             { return "POST" }
func (NewVoiceEndpoint) Path() string               { return "/api/voices/new" }
func (NewVoiceEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (NewVoiceEndpoint) Handle(ctx context.Context, req VoiceSampleReq) (VoiceSampleResp, error) {
	return speakSample(ctx, req, true)
}

// codeVoiceNotCustom refuses a sample or a new voice for a voice not set
// to custom.
const codeVoiceNotCustom = "voice_not_custom"

func speakSample(ctx context.Context, req VoiceSampleReq, renew bool) (VoiceSampleResp, error) {
	if req.Voice != string(engine.VoiceCustom) || !offersCustom(engine.Genre(req.Genre)) {
		return VoiceSampleResp{}, coded(http.StatusBadRequest, codeVoiceNotCustom, "a voice is heard and made only when it is set to custom")
	}
	if utf8.RuneCountInString(req.VoiceStyle) > maxVoiceText || utf8.RuneCountInString(req.Scenario) > maxVoiceText {
		return VoiceSampleResp{}, errBadRequest("the voice style or the scenario is longer than " + strconv.Itoa(maxVoiceText) + " characters")
	}
	sample, err := session.SpeakVoiceSample(ctx, session.VoiceSampleRequest{
		Genre: req.Genre, VoiceStyle: req.VoiceStyle, Scenario: req.Scenario, Renew: renew,
	})
	if err != nil {
		return VoiceSampleResp{}, err
	}
	return VoiceSampleResp{Audio: sample.Audio, ContentType: sample.ContentType, Stored: sample.Stored, OwnVoice: sample.Voice != ""}, nil
}

// offersCustom says whether a voice of the genre may be set to custom.
func offersCustom(genre engine.Genre) bool {
	return slices.ContainsFunc(genre.VoiceSlots(), func(slot engine.VoiceSlot) bool {
		return slices.Contains(slot.Choices, engine.VoiceCustom)
	})
}

// VoiceSampleReq is the form as the editor holds it: the game's genre, the
// choice for the voice to hear (custom, or it is refused), its voice style
// and its scenario, which describes a voice the style leaves open.
type VoiceSampleReq struct {
	Genre      string `json:"genre"`
	Voice      string `json:"voice"`
	VoiceStyle string `json:"voiceStyle"`
	Scenario   string `json:"scenario"`
}

// VoiceSampleResp is the sample: the audio (base64 in JSON) and its media
// type; stored when it was kept from an earlier preview; ownVoice when a
// voice made for the style speaks, not a built-in one.
type VoiceSampleResp struct {
	Audio       []byte `json:"audio"`
	ContentType string `json:"contentType"`
	Stored      bool   `json:"stored"`
	OwnVoice    bool   `json:"ownVoice"`
}
