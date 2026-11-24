package engine

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"engine/internal/adapters"
)

// An admin may replace the models of a configurable platform and set the
// facts the engine prices and compacts by, as providers retire models and
// change prices faster than releases come. The platform hands the engine its
// overrides (SetOverrides); every catalogue read applies them, so a session
// launched after a change runs on it, and one already running keeps what it
// settled on.

// ModelSlot is one cell of a platform's tier table: the model a tier runs a
// model type on.
type ModelSlot struct {
	Platform Platform  `json:"platform"`
	Tier     Tier      `json:"tier"`
	Type     ModelType `json:"type"`
}

// ModelFactsOverride is a model's entry in the model table as an admin sets
// it: what it takes and makes, its price, its context window (zero:
// unknown), and the day they set it. It stands for the whole entry: an empty
// In or Out keeps what the code's entry says, as for a model the admin only
// prices.
type ModelFactsOverride struct {
	In     []Medium `json:"in,omitempty"`
	Out    []Medium `json:"out,omitempty"`
	Price  Price    `json:"price"`
	Window int      `json:"window,omitempty"`
	Since  string   `json:"since"` // YYYY-MM-DD
}

// SlotChoice is what an admin assigns to a slot: its model, and for a
// picture its quality (empty keeps the slot's). The slot's other parameters,
// such as a picture's size or a voice, stay.
type SlotChoice struct {
	Model   string `json:"model"`
	Quality string `json:"quality,omitempty"`
}

// ImageQualities are the qualities an admin may draw a tier's pictures at,
// cheapest first.
func ImageQualities() []string { return []string{"low", "medium", "high"} }

// OverrideSource is the source a model's facts name when an admin set them.
const OverrideSource = "admin"

// Overrides are the admin's changes to the catalogue: models by slot, and
// facts by platform and model name. Only a configurable platform's apply.
type Overrides struct {
	Models map[ModelSlot]SlotChoice
	Facts  map[Platform]map[string]ModelFactsOverride
}

var (
	overridesMu sync.RWMutex
	overrides   Overrides
)

// SetOverrides replaces the admin's overrides, all at once, after
// CheckOverrides; refused, it keeps the ones it had.
func SetOverrides(o Overrides) error {
	copied, err := checked(o)
	if err != nil {
		return err
	}
	overridesMu.Lock()
	defer overridesMu.Unlock()
	overrides = copied
	return nil
}

// CheckOverrides says whether the engine takes these overrides, so the
// platform stores none it would refuse: each names a configurable platform,
// a known tier and model type, a model, a known picture quality (pictures
// only) and known media; and the code's
// tables with the overrides on them hold together (catalogueProblems):
// every model a tier is assigned is in the model table and fits its work, so
// a model in use cannot leave the table.
func CheckOverrides(o Overrides) error {
	_, err := checked(o)
	return err
}

// checked is a copy of o sharing nothing with the caller, or why the engine
// refuses it.
func checked(o Overrides) (Overrides, error) {
	configurable := map[Platform]bool{}
	for _, p := range defaultEntries() {
		configurable[Platform(p.ID)] = p.Configurable
	}
	copied := Overrides{Models: map[ModelSlot]SlotChoice{}, Facts: map[Platform]map[string]ModelFactsOverride{}}
	for slot, choice := range o.Models {
		switch {
		case !configurable[slot.Platform]:
			return Overrides{}, fmt.Errorf("platform %q is not configurable", slot.Platform)
		case !slot.Tier.Known():
			return Overrides{}, fmt.Errorf("unknown tier %q", slot.Tier)
		case !knownModelType(slot.Type):
			return Overrides{}, fmt.Errorf("unknown model type %q", slot.Type)
		case choice.Model == "":
			return Overrides{}, fmt.Errorf("%s %s %s: no model named", slot.Platform, slot.Tier, slot.Type)
		case choice.Quality != "" && slot.Type != ModelImage:
			return Overrides{}, fmt.Errorf("%s %s %s: only a picture has a quality", slot.Platform, slot.Tier, slot.Type)
		case choice.Quality != "" && !slices.Contains(ImageQualities(), choice.Quality):
			return Overrides{}, fmt.Errorf("%s %s %s: unknown quality %q", slot.Platform, slot.Tier, slot.Type, choice.Quality)
		}
		copied.Models[ModelSlot{slot.Platform, slot.Tier.Normal(), slot.Type}] = choice
	}
	for platform, facts := range o.Facts {
		if !configurable[platform] {
			return Overrides{}, fmt.Errorf("platform %q is not configurable", platform)
		}
		copied.Facts[platform] = map[string]ModelFactsOverride{}
		for model, f := range facts {
			if model == "" {
				return Overrides{}, fmt.Errorf("%s: facts for no model", platform)
			}
			for _, m := range append(slices.Clone(f.In), f.Out...) {
				if !knownMedium(m) {
					return Overrides{}, fmt.Errorf("%s %s: unknown medium %q", platform, model, m)
				}
			}
			f.In, f.Out = slices.Clone(f.In), slices.Clone(f.Out)
			copied.Facts[platform][model] = f
		}
	}
	for _, p := range defaultEntries() {
		if problems := catalogueProblems(copied.apply(p)); len(problems) > 0 {
			return Overrides{}, fmt.Errorf("%s", strings.Join(problems, "; "))
		}
	}
	return copied, nil
}

func currentOverrides() Overrides {
	overridesMu.RLock()
	defer overridesMu.RUnlock()
	return overrides
}

func knownMedium(m Medium) bool {
	return m == adapters.MediumText || m == adapters.MediumImage || m == adapters.MediumAudio
}

func knownModelType(t ModelType) bool {
	for _, known := range adapters.ModelTypes() {
		if known == t {
			return true
		}
	}
	return false
}

// apply is p with the overrides on it, sharing no map with the tables: a
// slot takes the admin's model, and a picture their quality, keeping its
// other parameters; a model's entry is the admin's,
// dated and sourced as theirs, keeping the code's media where the admin named
// none.
func (o Overrides) apply(p adapters.Platform) adapters.Platform {
	if !p.Configurable {
		return p
	}
	p = p.Clone()
	id := Platform(p.ID)
	for i, tier := range p.Tiers {
		for kind, model := range tier.Models {
			if choice, ok := o.Models[ModelSlot{id, tier.Tier, kind}]; ok {
				model.Name = choice.Model
				if choice.Quality != "" {
					if model.Params == nil {
						model.Params = map[string]string{}
					}
					model.Params[adapters.ParamQuality] = choice.Quality
				}
				p.Tiers[i].Models[kind] = model
			}
		}
	}
	for model, f := range o.Facts[id] {
		facts := p.Facts[model]
		if len(f.In) > 0 {
			facts.In = slices.Clone(f.In)
		}
		if len(f.Out) > 0 {
			facts.Out = slices.Clone(f.Out)
		}
		facts.Price, facts.Read, facts.Source = f.Price, f.Since, OverrideSource
		facts.Window, facts.WindowRead, facts.WindowSource = f.Window, f.Since, OverrideSource
		p.Facts[model] = facts
	}
	return p
}
