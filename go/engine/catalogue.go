package engine

import (
	"slices"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
)

// The catalogue's parts, re-exported so a caller can read them without importing internal/.
type (
	Toggle     = adapters.Toggle
	Toggles    = adapters.Toggles
	ModelType  = adapters.ModelType
	Model      = adapters.Model
	ModelSet   = adapters.ModelSet
	TierModels = adapters.TierModels
)

const (
	ToggleImages    = adapters.ToggleImages
	ToggleAudioOut  = adapters.ToggleAudioOut
	ToggleAudioIn   = adapters.ToggleAudioIn
	ToggleAudioLive = adapters.ToggleAudioLive

	ModelTooling       = adapters.ModelTooling
	ModelStory         = adapters.ModelStory
	ModelImage         = adapters.ModelImage
	ModelLive          = adapters.ModelLive
	ModelSpeech        = adapters.ModelSpeech
	ModelTranscription = adapters.ModelTranscription
)

// PlatformInfo is one AI platform the engine can run: what a person choosing a
// platform and a tier needs to know. Genres follow from Toggles.
type PlatformInfo struct {
	ID       Platform     `json:"id"`
	Name     string       `json:"name"`
	UserKeys bool         `json:"userKeys"`
	Toggles  Toggles      `json:"toggles"`
	Genres   []Genre      `json:"genres"`
	Tiers    []TierModels `json:"tiers"`
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
		})
	}
	return out
}

// AcceptsKeys reports whether the engine runs a platform on user API keys.
func AcceptsKeys(id Platform) bool {
	return slices.ContainsFunc(Platforms(), func(p PlatformInfo) bool { return p.ID == id && p.UserKeys })
}

// Genres is every genre the engine runs.
func Genres() []Genre { return []Genre{GenreAdventure, GenreNPCLive} }

// Needs is the toggles a genre cannot run without.
func (g Genre) Needs() []Toggle {
	switch g {
	case GenreNPCLive:
		return []Toggle{ToggleAudioLive, ToggleImages}
	}
	return nil
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
