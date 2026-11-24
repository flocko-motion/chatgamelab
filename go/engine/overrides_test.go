package engine

import (
	"context"
	"errors"
	"testing"

	"engine/internal/adapters"
	"engine/internal/ports"
)

func platformInfo(t *testing.T, infos []PlatformInfo, id Platform) PlatformInfo {
	t.Helper()
	for _, p := range infos {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("no platform %s", id)
	return PlatformInfo{}
}

func tierModels(t *testing.T, p PlatformInfo, tier Tier) ModelSet {
	t.Helper()
	for _, tm := range p.Tiers {
		if tm.Tier == tier {
			return tm.Models
		}
	}
	t.Fatalf("%s has no tier %s", p.ID, tier)
	return nil
}

// An admin's replaced model is what the catalogue lists in its slot, with the
// slot's parameters; the default catalogue keeps the code's; clearing the
// overrides brings the code's back.
func TestOverriddenModelReplacesItsSlotOnly(t *testing.T) {
	t.Cleanup(func() { _ = SetOverrides(Overrides{}) })
	defaultImage := tierModels(t, platformInfo(t, DefaultPlatforms(), PlatformOpenAI), TierLow)[ModelImage]

	err := SetOverrides(Overrides{
		Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, ModelImage}: {Model: "gpt-image-next"}},
		Facts: map[Platform]map[string]ModelFactsOverride{PlatformOpenAI: {"gpt-image-next": {
			In: []Medium{MediumText, MediumImage}, Out: []Medium{MediumImage}, Price: Price{OutputPerMTok: 30}, Since: "2026-11-24",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	openaiNow := platformInfo(t, Platforms(), PlatformOpenAI)
	low := tierModels(t, openaiNow, TierLow)[ModelImage]
	if low.Name != "gpt-image-next" || low.Params[adapters.ParamSize] != defaultImage.Params[adapters.ParamSize] {
		t.Errorf("low image = %+v, want gpt-image-next with the slot's params %+v", low, defaultImage.Params)
	}
	if medium := tierModels(t, openaiNow, TierMedium)[ModelImage]; medium.Name == "gpt-image-next" {
		t.Error("the override reached another tier")
	}
	if got := tierModels(t, platformInfo(t, DefaultPlatforms(), PlatformOpenAI), TierLow)[ModelImage]; got.Name != defaultImage.Name {
		t.Errorf("default catalogue changed: %s", got.Name)
	}
	entry, _ := catalogueEntry(PlatformOpenAI)
	if set, _ := entry.Models(TierLow); set[ModelImage].Name != "gpt-image-next" {
		t.Error("sessions do not run on the override")
	}

	_ = SetOverrides(Overrides{})
	if got := tierModels(t, platformInfo(t, Platforms(), PlatformOpenAI), TierLow)[ModelImage]; got.Name != defaultImage.Name {
		t.Errorf("after reset: %s, want %s", got.Name, defaultImage.Name)
	}
}

// An admin's model entry prices a session's calls: an unpriced model becomes
// priced, and its window is what a thread compacts by.
func TestOverriddenFactsPriceAndWindow(t *testing.T) {
	t.Cleanup(func() { _ = SetOverrides(Overrides{}) })
	err := SetOverrides(Overrides{Facts: map[Platform]map[string]ModelFactsOverride{
		PlatformOpenAI: {"gpt-new": {In: []Medium{MediumText}, Out: []Medium{MediumText},
			Price: Price{InputPerMTok: 1, OutputPerMTok: 2}, Window: 400_000, Since: "2026-11-24"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	facts := platformInfo(t, Platforms(), PlatformOpenAI).Models["gpt-new"]
	if facts.Source != OverrideSource || facts.Read != "2026-11-24" || facts.Window != 400_000 {
		t.Errorf("facts = %+v", facts)
	}

	ledger := newUsageLedger()
	ledger.price(PlatformOpenAI)
	ledger.record(ports.Usage{Node: "n", Model: "gpt-new", InputTokens: 1_000_000, OutputTokens: 1_000_000})
	report := ledger.report()
	if !report.Complete || report.TotalCost != 3 {
		t.Errorf("report = complete %v, cost %v; want priced at 3", report.Complete, report.TotalCost)
	}
	if ledger.window("gpt-new") != 400_000 {
		t.Errorf("window = %d", ledger.window("gpt-new"))
	}
}

// Only a configurable platform takes overrides; a refused set keeps the
// overrides there were.
func TestOverridesOnlyForConfigurablePlatforms(t *testing.T) {
	t.Cleanup(func() { _ = SetOverrides(Overrides{}) })
	if !platformInfo(t, Platforms(), PlatformOpenAI).Configurable || platformInfo(t, Platforms(), PlatformMistral).Configurable {
		t.Fatal("OpenAI is configurable, Mistral is not")
	}
	kept := Overrides{Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierHigh, ModelStory}: {Model: "gpt-5.6-luna"}}}
	if err := SetOverrides(kept); err != nil {
		t.Fatal(err)
	}
	for name, o := range map[string]Overrides{
		"mistral model":   {Models: map[ModelSlot]SlotChoice{{PlatformMistral, TierLow, ModelStory}: {Model: "m"}}},
		"mistral facts":   {Facts: map[Platform]map[string]ModelFactsOverride{PlatformMistral: {"m": {}}}},
		"unknown tier":    {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, "ultra", ModelStory}: {Model: "m"}}},
		"unknown type":    {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, "dream"}: {Model: "m"}}},
		"empty name":      {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, ModelStory}: {Model: ""}}},
		"quality on text": {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, ModelStory}: {Model: "gpt-5.6-luna", Quality: "low"}}},
		"unknown quality": {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, ModelImage}: {Model: "gpt-image-2.5-flare", Quality: "ultra"}}},
		"bad medium":      {Facts: map[Platform]map[string]ModelFactsOverride{PlatformOpenAI: {"m": {In: []Medium{"smell"}}}}},
		// an assignment names a model in the model table, fitting its work
		"model not in the table":   {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, ModelStory}: {Model: "gpt-nowhere"}}},
		"model unfit for the work": {Models: map[ModelSlot]SlotChoice{{PlatformOpenAI, TierLow, ModelHearing}: {Model: "gpt-6.2-sol"}}},
		"story model loses its window": {Facts: map[Platform]map[string]ModelFactsOverride{PlatformOpenAI: {"gpt-6.2-sol": {
			Price: Price{InputPerMTok: 1}, Since: "2026-11-24"}}}},
	} {
		if err := SetOverrides(o); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if got := tierModels(t, platformInfo(t, Platforms(), PlatformOpenAI), TierHigh)[ModelStory]; got.Name != "gpt-5.6-luna" {
		t.Errorf("a refused set dropped the kept override: %s", got.Name)
	}
}

// A probe answers on the mock for text and image models, and says it cannot
// probe the others yet.
func TestProbeModel(t *testing.T) {
	offer := Offer{ID: "mock", Platform: PlatformMock}
	for _, kind := range []ModelType{ModelTooling, ModelStory, ModelImage} {
		if _, err := ProbeModel(context.Background(), offer, kind, "mock-model"); err != nil {
			t.Errorf("%s: %v", kind, err)
		}
	}
	for _, kind := range []ModelType{ModelLive, ModelSpeech, ModelHearing} {
		if _, err := ProbeModel(context.Background(), offer, kind, "mock-model"); !errors.Is(err, ErrNoProbe) {
			t.Errorf("%s: %v, want ErrNoProbe", kind, err)
		}
	}
	if _, err := ProbeModel(context.Background(), Offer{ID: "k", Platform: PlatformOpenAI}, ModelTooling, "gpt-x"); err == nil {
		t.Error("probing without a key went through")
	}
}

// A picture's quality is the admin's to set per tier; its size stays the slot's.
func TestOverriddenPictureQuality(t *testing.T) {
	t.Cleanup(func() { _ = SetOverrides(Overrides{}) })
	defaultImage := tierModels(t, platformInfo(t, DefaultPlatforms(), PlatformOpenAI), TierHigh)[ModelImage]
	err := SetOverrides(Overrides{Models: map[ModelSlot]SlotChoice{
		{PlatformOpenAI, TierHigh, ModelImage}: {Model: defaultImage.Name, Quality: "medium"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := tierModels(t, platformInfo(t, Platforms(), PlatformOpenAI), TierHigh)[ModelImage]
	if got.Params[adapters.ParamQuality] != "medium" || got.Params[adapters.ParamSize] != defaultImage.Params[adapters.ParamSize] {
		t.Errorf("high image = %+v, want medium at the slot's size", got)
	}
}
