package engine

import (
	"maps"
	"slices"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/genre"
)

// The catalogue's parts, re-exported so a caller can read them without importing internal/.
type (
	Toggle     = adapters.Toggle
	Toggles    = adapters.Toggles
	ModelType  = adapters.ModelType
	Model      = adapters.Model
	ModelSet   = adapters.ModelSet
	TierModels = adapters.TierModels
	Medium     = adapters.Medium
	ModelFacts = adapters.ModelFacts
	ModelTable = adapters.ModelTable
	Price      = adapters.Price
)

const (
	ToggleImages    = adapters.ToggleImages
	ToggleAudioOut  = adapters.ToggleAudioOut
	ToggleAudioIn   = adapters.ToggleAudioIn
	ToggleAudioLive = adapters.ToggleAudioLive

	ModelTooling = adapters.ModelTooling
	ModelStory   = adapters.ModelStory
	ModelImage   = adapters.ModelImage
	ModelLive    = adapters.ModelLive
	ModelSpeech  = adapters.ModelSpeech
	ModelHearing = adapters.ModelHearing
)

// PlatformInfo is one AI platform the engine can run: what a person choosing a
// platform and a tier needs to know. Genres follow from Toggles. Models says
// what each model is: what it takes and makes, and its price, dated, with its
// source.
type PlatformInfo struct {
	ID       Platform     `json:"id"`
	Name     string       `json:"name"`
	UserKeys bool         `json:"userKeys"`
	Toggles  Toggles      `json:"toggles"`
	Genres   []Genre      `json:"genres"`
	Tiers    []TierModels `json:"tiers"`
	Models   ModelTable   `json:"models"`
}

// catalogueEntries is every platform the engine runs, the mock last.
func catalogueEntries() []adapters.Platform {
	return []adapters.Platform{openai.Platform(), mistral.Platform(), mock.Platform()}
}

// Platforms is the engine's catalogue, a copy the caller may change. The mock
// comes last; whoever lists it decides when a person sees it.
func Platforms() []PlatformInfo {
	var out []PlatformInfo
	for _, p := range catalogueEntries() {
		p = p.Clone()
		out = append(out, PlatformInfo{
			ID:       Platform(p.ID),
			Name:     p.Name,
			UserKeys: p.UserKeys,
			Toggles:  p.Toggles,
			Genres:   PlayableGenres(p.Toggles),
			Tiers:    p.Tiers,
			Models:   p.Facts,
		})
	}
	return out
}

// AcceptsKeys reports whether the engine runs a platform on user API keys.
func AcceptsKeys(id Platform) bool {
	return slices.ContainsFunc(Platforms(), func(p PlatformInfo) bool { return p.ID == id && p.UserKeys })
}

// Genres is every genre the engine runs.
func Genres() []Genre { return []Genre{GenreAdventure, GenreNPCLive, GenreNPCDuo} }

// Needs is the toggles a genre cannot run without.
func (g Genre) Needs() []Toggle {
	switch g {
	case GenreNPCLive, GenreNPCDuo:
		return []Toggle{ToggleAudioLive}
	}
	return nil
}

// GameField is a field of a game that a genre reads, by the name a client
// edits it under.
type GameField string

const (
	FieldScenario     GameField = "scenario"
	FieldGameStart    GameField = "gameStart"
	FieldImageStyle   GameField = "imageStyle"
	FieldVoiceStyle   GameField = "voiceStyle"
	FieldStatusFields GameField = "statusFields"
)

// Fields is the game's fields the genre reads, so an editor offers those
// and no others: an Adventure tells a world in pictures and status, and may
// be narrated; NPC-Live is one character's voice. Every genre has the image
// style, as every game's cover is drawn in it.
func (g Genre) Fields() []GameField {
	switch g {
	case GenreNPCLive:
		return []GameField{FieldScenario, FieldGameStart, FieldImageStyle, FieldVoiceStyle}
	case GenreNPCDuo:
		// Both characters share the scenario field; their voices are cast
		// from who they are.
		return []GameField{FieldScenario, FieldGameStart, FieldImageStyle}
	}
	return []GameField{FieldScenario, FieldGameStart, FieldImageStyle, FieldStatusFields, FieldVoiceStyle}
}

// GenrePrompt is one of a genre's own prompts as it is written: a template
// whose fields are filled for each session, which a spec's override may use
// too.
type GenrePrompt struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

// Prompts are the genre's own prompts, by name, sorted: what a session runs
// on unless its spec replaces one.
func (g Genre) Prompts() []GenrePrompt {
	defaults := genre.AdventurePrompts
	switch g {
	case GenreNPCLive:
		defaults = genre.NPCLivePrompts
	case GenreNPCDuo:
		defaults = genre.NPCDuoPrompts
	}
	out := []GenrePrompt{}
	for _, name := range slices.Sorted(maps.Keys(defaults)) {
		out = append(out, GenrePrompt{Name: name, Text: defaults[name]})
	}
	return out
}

// MakesVoices reports whether the genre makes a voice of its own for a game,
// kept per key and voice style: whether its wiring holds a block that makes
// one. A genre that does not speaks in a provider's built-in voice, which the
// voice style only steers, so there is no voice to remake.
func (g Genre) MakesVoices() bool {
	s := &Session{usage: newUsageLedger()}
	w, err := s.wire(SessionSpec{Genre: g})
	return err == nil && w.Graph.MakesVoices()
}

// PlayableGenres is every genre whose needs the toggles meet.
func PlayableGenres(toggles Toggles) []Genre {
	var out []Genre
	for _, g := range Genres() {
		if missing(g, toggles) == nil {
			out = append(out, g)
		}
	}
	return out
}

func missing(g Genre, toggles Toggles) []Toggle {
	var out []Toggle
	for _, need := range g.Needs() {
		if !toggles.Has(need) {
			out = append(out, need)
		}
	}
	return out
}
