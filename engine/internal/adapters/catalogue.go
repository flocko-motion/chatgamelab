package adapters

import (
	"fmt"
	"maps"
)

// Tier is the session's quality setting. It decides how good a model is,
// never whether a feature exists.
type Tier string

const (
	TierLow    Tier = "low"
	TierMedium Tier = "medium"
	TierHigh   Tier = "high"
)

// tierMax is not offered; stored data and specs that name it mean high.
const tierMax Tier = "max"

// Tiers is every tier on offer, cheapest first.
func Tiers() []Tier { return []Tier{TierLow, TierMedium, TierHigh} }

// Normal reads a retired name as the tier it stands for.
func (t Tier) Normal() Tier {
	if t == tierMax {
		return TierHigh
	}
	return t
}

// UnmarshalText applies Normal, so decoded data never carries a retired name.
func (t *Tier) UnmarshalText(text []byte) error {
	*t = Tier(text).Normal()
	return nil
}

// Label is the tier's name for a person choosing one.
func (t Tier) Label() string {
	switch t.Normal() {
	case TierLow:
		return "Economy"
	case TierMedium:
		return "Balanced"
	case TierHigh:
		return "High"
	}
	return string(t)
}

// ModelType is a kind of model a tier lists.
type ModelType string

const (
	ModelTooling       ModelType = "tooling"
	ModelStory         ModelType = "story"
	ModelImage         ModelType = "image"
	ModelLive          ModelType = "live"
	ModelSpeech        ModelType = "speech"
	ModelTranscription ModelType = "transcription"
)

// ModelTypes is every model type, in display order.
func ModelTypes() []ModelType {
	return []ModelType{ModelTooling, ModelStory, ModelImage, ModelLive, ModelSpeech, ModelTranscription}
}

// Model parameter names.
const (
	ParamSize    = "size"
	ParamQuality = "quality"
	ParamVoice   = "voice"
	ParamFormat  = "format"
)

// Model is one model and the parameters it is called with.
type Model struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
}

// ModelSet is one tier's model per type; a missing type is a feature the
// platform does not have.
type ModelSet map[ModelType]Model

// Toggle names a feature a platform has or lacks.
type Toggle string

const (
	ToggleImages    Toggle = "images"
	ToggleAudioOut  Toggle = "audio-out"
	ToggleAudioIn   Toggle = "audio-in"
	ToggleAudioLive Toggle = "audio-live"
)

// AllToggles is every toggle, in display order.
func AllToggles() []Toggle {
	return []Toggle{ToggleImages, ToggleAudioOut, ToggleAudioIn, ToggleAudioLive}
}

// ModelType is the model a toggle's feature runs on.
func (t Toggle) ModelType() ModelType {
	switch t {
	case ToggleImages:
		return ModelImage
	case ToggleAudioOut:
		return ModelSpeech
	case ToggleAudioIn:
		return ModelTranscription
	case ToggleAudioLive:
		return ModelLive
	}
	return ""
}

// Toggles states each feature for one platform.
type Toggles struct {
	Images    bool `json:"images"`
	AudioOut  bool `json:"audio-out"`
	AudioIn   bool `json:"audio-in"`
	AudioLive bool `json:"audio-live"`
}

// Has reports whether a feature is on.
func (t Toggles) Has(toggle Toggle) bool {
	switch toggle {
	case ToggleImages:
		return t.Images
	case ToggleAudioOut:
		return t.AudioOut
	case ToggleAudioIn:
		return t.AudioIn
	case ToggleAudioLive:
		return t.AudioLive
	}
	return false
}

// ModelTypes is what every tier must list: tooling and story, plus one type
// per toggle that is on.
func (t Toggles) ModelTypes() []ModelType {
	enabled := map[ModelType]bool{ModelTooling: true, ModelStory: true}
	for _, toggle := range AllToggles() {
		enabled[toggle.ModelType()] = t.Has(toggle)
	}
	var types []ModelType
	for _, kind := range ModelTypes() {
		if enabled[kind] {
			types = append(types, kind)
		}
	}
	return types
}

// TierModels is one tier of a platform.
type TierModels struct {
	Tier   Tier     `json:"id"`
	Label  string   `json:"label"`
	Models ModelSet `json:"models"`
}

// Ladder lists a platform's tiers, cheapest first.
func Ladder(low, medium, high ModelSet) []TierModels {
	sets := []ModelSet{low, medium, high}
	tiers := make([]TierModels, len(sets))
	for i, tier := range Tiers() {
		tiers[i] = TierModels{Tier: tier, Label: tier.Label(), Models: sets[i]}
	}
	return tiers
}

// Platform is what a person choosing a platform and a tier needs to know.
type Platform struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	UserKeys bool         `json:"userKeys"`
	Toggles  Toggles      `json:"toggles"`
	Tiers    []TierModels `json:"tiers"`
}

// Models is one tier's model set, a copy the caller may change.
func (p Platform) Models(tier Tier) (ModelSet, error) {
	tier = tier.Normal()
	for _, t := range p.Tiers {
		if t.Tier == tier {
			return maps.Clone(t.Models), nil
		}
	}
	return nil, fmt.Errorf("%s: unknown tier %q", p.ID, tier)
}
