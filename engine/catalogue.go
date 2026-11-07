package engine

import (
	"fmt"
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

// Platforms is the engine's catalogue. The mock comes last; whoever lists it
// decides when a person sees it.
func Platforms() []PlatformInfo {
	var out []PlatformInfo
	for _, p := range []adapters.Platform{openai.Platform(), mistral.Platform(), mock.Platform()} {
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

// platformInfo finds a platform in the catalogue; empty means the mock.
func platformInfo(id Platform) (PlatformInfo, error) {
	if id == "" {
		id = PlatformMock
	}
	all := Platforms()
	i := slices.IndexFunc(all, func(p PlatformInfo) bool { return p.ID == id })
	if i < 0 {
		return PlatformInfo{}, fmt.Errorf("unknown platform %q", id)
	}
	return all[i], nil
}

// playable refuses a genre the platform lacks a feature for, before anything is built.
func (spec SessionSpec) playable() error {
	if !slices.Contains(Genres(), spec.Genre) {
		return fmt.Errorf("unknown genre %q", spec.Genre)
	}
	p, err := platformInfo(spec.Platform)
	if err != nil {
		return err
	}
	if lacks := missing(spec.Genre, p.Toggles); lacks != nil {
		return fmt.Errorf("platform %q cannot run %s: it lacks %v", p.ID, spec.Genre, lacks)
	}
	return nil
}
