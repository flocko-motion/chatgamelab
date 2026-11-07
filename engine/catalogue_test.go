package engine

import (
	"slices"
	"testing"

	"engine/internal/adapters"
)

func TestGenresFollowFromToggles(t *testing.T) {
	cases := []struct {
		name    string
		toggles Toggles
		want    []Genre
	}{
		{"nothing on", Toggles{}, []Genre{GenreAdventure}},
		{"images only", Toggles{Images: true}, []Genre{GenreAdventure}},
		{"live only", Toggles{AudioLive: true}, []Genre{GenreAdventure}},
		{"live and images", Toggles{AudioLive: true, Images: true}, []Genre{GenreAdventure, GenreNPCLive}},
		{"everything", Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true}, []Genre{GenreAdventure, GenreNPCLive}},
	}
	for _, c := range cases {
		if got := PlayableGenres(c.toggles); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCatalogueListsItsPlatforms(t *testing.T) {
	var ids []Platform
	for _, p := range Platforms() {
		ids = append(ids, p.ID)
	}
	if want := []Platform{PlatformOpenAI, PlatformMistral, PlatformMock}; !slices.Equal(ids, want) {
		t.Errorf("platforms = %v, want %v", ids, want)
	}

	mock := Platforms()[2]
	if mock.UserKeys {
		t.Error("the mock takes no key")
	}
	if mock.Toggles != (Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true}) {
		t.Errorf("mock toggles = %+v, want every one on", mock.Toggles)
	}
}

// Tiers decide quality, never whether a feature exists: every tier of every
// platform names a model for exactly the types its toggles enable.
func TestEveryTierNamesAModelPerEnabledType(t *testing.T) {
	for _, p := range Platforms() {
		var tiers []Tier
		for _, tier := range p.Tiers {
			tiers = append(tiers, tier.Tier)
			if tier.Label != tier.Tier.Label() {
				t.Errorf("%s/%s: label %q", p.ID, tier.Tier, tier.Label)
			}

			want := p.Toggles.ModelTypes()
			for _, kind := range want {
				if tier.Models[kind].Name == "" {
					t.Errorf("%s/%s: no %s model", p.ID, tier.Tier, kind)
				}
			}
			for kind := range tier.Models {
				if !slices.Contains(want, kind) {
					t.Errorf("%s/%s: lists a %s model its toggles leave off", p.ID, tier.Tier, kind)
				}
			}
		}
		if !slices.Equal(tiers, adapters.Tiers()) {
			t.Errorf("%s: tiers = %v, want %v", p.ID, tiers, adapters.Tiers())
		}
	}
}

func TestAcceptsKeys(t *testing.T) {
	for id, want := range map[Platform]bool{
		PlatformOpenAI:  true,
		PlatformMistral: true,
		PlatformMock:    false,
		"ionos":         false,
		"":              false,
	} {
		if got := AcceptsKeys(id); got != want {
			t.Errorf("AcceptsKeys(%q) = %v, want %v", id, got, want)
		}
	}
}

// A caller holds a copy: changing what Platforms returned leaves the tier
// tables, and the next caller's copy, as they were.
func TestPlatformsAreCopies(t *testing.T) {
	for _, p := range Platforms() {
		for _, tier := range p.Tiers {
			for kind, model := range tier.Models {
				for name := range model.Params {
					model.Params[name] = "tampered"
				}
				model.Name = "tampered"
				tier.Models[kind] = model
			}
		}
	}

	for _, p := range Platforms() {
		for _, tier := range p.Tiers {
			for kind, model := range tier.Models {
				if model.Name == "tampered" {
					t.Errorf("%s/%s/%s: name changed through a copy", p.ID, tier.Tier, kind)
				}
				for name, value := range model.Params {
					if value == "tampered" {
						t.Errorf("%s/%s/%s: param %s changed through a copy", p.ID, tier.Tier, kind, name)
					}
				}
			}
		}
	}

	entry, _ := catalogueEntry(PlatformOpenAI)
	models, err := entry.Models(TierLow)
	if err != nil {
		t.Fatal(err)
	}
	models[ModelSpeech].Params[adapters.ParamVoice] = "tampered"
	again, _ := entry.Models(TierLow)
	if again[ModelSpeech].Params[adapters.ParamVoice] == "tampered" {
		t.Error("a tier's model set shares its parameters with the table")
	}
}
