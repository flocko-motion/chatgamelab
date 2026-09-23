package engine

import (
	"strings"
	"testing"

	"engine/internal/adapters"
)

// A stand-in platform table, so these tests check the resolution rules rather
// than any particular vendor's model names.
func fakeTiers(t Tier) (adapters.ModelSet, error) {
	switch t {
	case TierEconomy:
		return adapters.ModelSet{Live: "live-eco", Tool: "tool-eco", Threaded: "thr-eco"}, nil
	case TierBalanced:
		return adapters.ModelSet{Live: "live-bal", Tool: "tool-bal", Threaded: "thr-bal"}, nil
	case TierMax:
		return adapters.ModelSet{Live: "live-max", Tool: "tool-max", Threaded: "thr-max"}, nil
	}
	return adapters.ModelSet{}, errUnknownTier(t)
}

func errUnknownTier(t Tier) error { return &tierError{t} }

type tierError struct{ t Tier }

func (e *tierError) Error() string { return "unknown tier " + string(e.t) }

func TestUnsetTierIsBalanced(t *testing.T) {
	got, err := SessionSpec{}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got.Threaded != "thr-bal" {
		t.Errorf("got %q, want the balanced default", got.Threaded)
	}
}

func TestModelTierSetsEveryUnpinnedRole(t *testing.T) {
	got, err := SessionSpec{ModelTier: TierMax}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got.Live != "live-max" || got.Tool != "tool-max" || got.Threaded != "thr-max" {
		t.Errorf("got %+v, want every role at max", got)
	}
}

// A literal pins one role without disturbing the rest.
func TestLiteralPinOverridesOneRole(t *testing.T) {
	got, err := SessionSpec{
		ModelTier: TierEconomy,
		ModelLive: "gpt-realtime-2.1",
	}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got.Live != "gpt-realtime-2.1" {
		t.Errorf("live = %q, want the literal", got.Live)
	}
	if got.Tool != "tool-eco" {
		t.Errorf("tool = %q, want economy to still apply", got.Tool)
	}
}

// "$tier" lifts one role to another tier, which is what lets a cheap session
// keep one expensive stage.
func TestSigilPinTakesOneRoleFromAnotherTier(t *testing.T) {
	got, err := SessionSpec{
		ModelTier:     TierEconomy,
		ModelThreaded: "$max",
	}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got.Threaded != "thr-max" {
		t.Errorf("threaded = %q, want the max tier's model", got.Threaded)
	}
	if got.Live != "live-eco" {
		t.Errorf("live = %q, want economy elsewhere", got.Live)
	}
}

// A model name is never mistaken for a tier, and a tier is never mistaken for a
// model name. That is the whole job of the sigil.
func TestSigilDistinguishesTierFromModelName(t *testing.T) {
	got, err := SessionSpec{ModelTool: "max"}.resolveModels(fakeTiers)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tool != "max" {
		t.Errorf("tool = %q; without the sigil it is a model name", got.Tool)
	}
}

func TestUnknownTierIsRefused(t *testing.T) {
	if _, err := (SessionSpec{ModelTier: "luxurious"}).resolveModels(fakeTiers); err == nil {
		t.Error("expected an unknown ModelTier to fail")
	}

	_, err := SessionSpec{ModelLive: "$luxurious"}.resolveModels(fakeTiers)
	if err == nil {
		t.Fatal("expected an unknown tier in a pin to fail")
	}
	if !strings.Contains(err.Error(), "modelLive") {
		t.Errorf("error should name the offending field, got: %v", err)
	}
}
