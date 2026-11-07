package engine

import (
	"encoding/json"
	"strings"
	"testing"

	"engine/internal/adapters"
)

// A stand-in platform table, so these tests check the resolution rules rather
// than any particular vendor's model names.
func fakeTiers(t Tier) (adapters.ModelSet, error) {
	set := func(suffix string) adapters.ModelSet {
		return adapters.ModelSet{
			ModelLive:    {Name: "live-" + suffix},
			ModelTooling: {Name: "tool-" + suffix},
			ModelStory:   {Name: "story-" + suffix},
			ModelImage:   {Name: "image-" + suffix, Params: map[string]string{adapters.ParamSize: suffix}},
		}
	}
	switch t {
	case TierLow:
		return set("low"), nil
	case TierMedium:
		return set("med"), nil
	case TierHigh:
		return set("high"), nil
	}
	return nil, &tierError{t}
}

type tierError struct{ t Tier }

func (e *tierError) Error() string { return "unknown tier " + string(e.t) }

func TestUnsetTierIsMedium(t *testing.T) {
	got, err := SessionSpec{}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got[ModelStory].Name != "story-med" {
		t.Errorf("got %q, want the medium default", got[ModelStory].Name)
	}
}

func TestModelTierSetsEveryUnpinnedType(t *testing.T) {
	got, err := SessionSpec{ModelTier: TierHigh}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got[ModelLive].Name != "live-high" || got[ModelTooling].Name != "tool-high" || got[ModelStory].Name != "story-high" {
		t.Errorf("got %+v, want every type at high", got)
	}
}

// A literal pins one type without disturbing the rest, and keeps the tier's parameters.
func TestLiteralPinOverridesOneType(t *testing.T) {
	got, err := SessionSpec{
		ModelTier: TierLow,
		Models:    map[ModelType]string{ModelLive: "gpt-live-1", ModelImage: "other-image"},
	}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got[ModelLive].Name != "gpt-live-1" {
		t.Errorf("live = %q, want the literal", got[ModelLive].Name)
	}
	if got[ModelTooling].Name != "tool-low" {
		t.Errorf("tooling = %q, want low to still apply", got[ModelTooling].Name)
	}
	if got[ModelImage].Params[adapters.ParamSize] != "low" {
		t.Errorf("image params = %v, want the tier's", got[ModelImage].Params)
	}
}

// "$tier" lifts one type to another tier, which is what lets a cheap session
// keep one expensive stage.
func TestSigilPinTakesOneTypeFromAnotherTier(t *testing.T) {
	got, err := SessionSpec{
		ModelTier: TierLow,
		Models:    map[ModelType]string{ModelStory: "$high"},
	}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got[ModelStory].Name != "story-high" {
		t.Errorf("story = %q, want the high tier's model", got[ModelStory].Name)
	}
	if got[ModelLive].Name != "live-low" {
		t.Errorf("live = %q, want low elsewhere", got[ModelLive].Name)
	}
}

// A model name is never mistaken for a tier, and a tier is never mistaken for a
// model name. That is the whole job of the sigil.
func TestSigilDistinguishesTierFromModelName(t *testing.T) {
	got, err := SessionSpec{Models: map[ModelType]string{ModelTooling: "high"}}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got[ModelTooling].Name != "high" {
		t.Errorf("tooling = %q; without the sigil it is a model name", got[ModelTooling].Name)
	}
}

// max is not offered, but stored data and specs that name it mean high.
func TestMaxIsReadAsHigh(t *testing.T) {
	var spec SessionSpec
	if err := json.Unmarshal([]byte(`{"modelTier":"max","models":{"tooling":"$max"}}`), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.ModelTier != TierHigh {
		t.Errorf("decoded tier = %q, want high", spec.ModelTier)
	}

	got, err := spec.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got[ModelStory].Name != "story-high" || got[ModelTooling].Name != "tool-high" {
		t.Errorf("got %+v, want high from the tier and from the pin", got)
	}

	if got, err := (SessionSpec{ModelTier: "max"}).resolveModels(fakeTiers); err != nil || got[ModelStory].Name != "story-high" {
		t.Errorf("unnormalised max: got %+v, %v; want high", got, err)
	}
}

func TestUnknownTierIsRefused(t *testing.T) {
	if _, err := (SessionSpec{ModelTier: "luxurious"}).resolveModels(fakeTiers); err == nil {
		t.Error("expected an unknown ModelTier to fail")
	}

	_, err := SessionSpec{Models: map[ModelType]string{ModelLive: "$luxurious"}}.resolveModels(fakeTiers)
	if err == nil {
		t.Fatal("expected an unknown tier in a pin to fail")
	}
	if !strings.Contains(err.Error(), "models.live") {
		t.Errorf("error should name the offending pin, got: %v", err)
	}
}

func TestUnknownModelTypeIsRefused(t *testing.T) {
	_, err := SessionSpec{Models: map[ModelType]string{"audio": "x"}}.resolveModels(fakeTiers)
	if err == nil {
		t.Fatal("expected an unknown model type to fail, so a typo cannot pass silently")
	}
}

// A sigil pin to a type the other tier lacks is refused rather than emptied.
func TestSigilPinToMissingTypeIsRefused(t *testing.T) {
	_, err := SessionSpec{Models: map[ModelType]string{ModelSpeech: "$high"}}.resolveModels(fakeTiers)
	if err == nil {
		t.Fatal("expected a pin to a missing speech model to fail")
	}
}
