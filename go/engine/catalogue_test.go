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
		// The live genre's picture comes from the gallery, so it needs no image model.
		{"live only", Toggles{AudioLive: true}, []Genre{GenreAdventure, GenreNPCLive, GenreNPCDuo}},
		{"live and images", Toggles{AudioLive: true, Images: true}, []Genre{GenreAdventure, GenreNPCLive, GenreNPCDuo}},
		{"everything", Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true}, []Genre{GenreAdventure, GenreNPCLive, GenreNPCDuo}},
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

// Every model a tier names has an entry in its platform's model table, so
// every model is priced and its media are known.
func TestEveryTierModelHasFacts(t *testing.T) {
	for _, p := range Platforms() {
		for _, tier := range p.Tiers {
			for kind, model := range tier.Models {
				facts, known := p.Models[model.Name]
				if !known {
					t.Errorf("%s/%s/%s: %s has no entry in the model table", p.ID, tier.Tier, kind, model.Name)
					continue
				}
				if facts.Read == "" || facts.Source == "" {
					t.Errorf("%s: %s's price says neither when nor where it was read", p.ID, model.Name)
				}
			}
		}
	}
}

// What a model type does needs what its model takes and makes: the hearing
// model takes audio, the speech model makes it, the image model makes
// pictures, and every model answers in text unless it only draws.
func TestTierModelsTakeAndMakeWhatTheirTypeNeeds(t *testing.T) {
	needs := map[ModelType]struct{ in, out adapters.Medium }{
		ModelTooling: {adapters.MediumText, adapters.MediumText},
		ModelStory:   {adapters.MediumText, adapters.MediumText},
		ModelImage:   {adapters.MediumText, adapters.MediumImage},
		ModelSpeech:  {adapters.MediumText, adapters.MediumAudio},
		ModelHearing: {adapters.MediumAudio, adapters.MediumText},
		ModelLive:    {adapters.MediumAudio, adapters.MediumAudio},
	}
	for _, p := range Platforms() {
		for _, tier := range p.Tiers {
			for kind, model := range tier.Models {
				facts, need := p.Models[model.Name], needs[kind]
				if !facts.Takes(need.in) || !facts.Makes(need.out) {
					t.Errorf("%s/%s: %s model %s takes %v and makes %v, want it to take %s and make %s",
						p.ID, tier.Tier, kind, model.Name, facts.In, facts.Out, need.in, need.out)
				}
			}
		}
	}
}

// Every model a story thread runs on, the story model and the speech model,
// has a context window, so the thread compacts before it overflows; the
// window says when and where it was read.
func TestThreadModelsHaveAContextWindow(t *testing.T) {
	for _, p := range Platforms() {
		for _, tier := range p.Tiers {
			for _, kind := range []ModelType{ModelStory, ModelSpeech} {
				model, listed := tier.Models[kind]
				if !listed {
					continue
				}
				facts := p.Models[model.Name]
				if facts.Window <= 0 {
					t.Errorf("%s/%s: %s model %s has no context window", p.ID, tier.Tier, kind, model.Name)
				}
				if facts.WindowRead == "" || facts.WindowSource == "" {
					t.Errorf("%s: %s's window says neither when nor where it was read", p.ID, model.Name)
				}
			}
		}
	}
}

// An editor offers a genre's own fields: NPC-Live is one character's voice,
// with no status, and the image style its cover is drawn in; an Adventure
// has all of them, and its narrator a voice.
func TestGenresNameTheirFields(t *testing.T) {
	fields := map[Genre][]GameField{}
	for _, g := range GenreList() {
		fields[g.ID] = g.Fields
	}
	npc := fields[GenreNPCLive]
	if !slices.Contains(npc, FieldVoiceStyle) || !slices.Contains(npc, FieldImageStyle) || slices.Contains(npc, FieldStatusFields) {
		t.Errorf("npc-live fields: %v", npc)
	}
	adventure := fields[GenreAdventure]
	for _, want := range []GameField{FieldScenario, FieldGameStart, FieldImageStyle, FieldStatusFields, FieldVoiceStyle} {
		if !slices.Contains(adventure, want) {
			t.Errorf("adventure lacks %q: %v", want, adventure)
		}
	}
}

// A genre makes voices of its own when its wiring holds a block that makes
// one: NPC-Live's character has one, an Adventure narrates in a built-in
// voice. The editor offers making a new voice only where one is made.
func TestGenreSaysWhetherItMakesVoices(t *testing.T) {
	want := map[Genre]bool{GenreAdventure: false, GenreNPCLive: true}
	for g, makes := range want {
		if got := g.MakesVoices(); got != makes {
			t.Errorf("%s.MakesVoices() = %v, want %v", g, got, makes)
		}
	}
	for _, info := range GenreList() {
		if info.MakesVoices != want[info.ID] {
			t.Errorf("GenreList: %s makesVoices = %v, want %v", info.ID, info.MakesVoices, want[info.ID])
		}
	}
}
