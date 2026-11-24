package engine

import (
	"reflect"
	"slices"
	"testing"

	"engine/internal/adapters"
)

// A genre holds a live and a rounds half, which play the same game; a genre
// may offer one of them for now.
func TestGenresComeInPairs(t *testing.T) {
	if got, want := Genres(), []Genre{GenreAdventure, GenreNPC, GenreDiscussion}; !slices.Equal(got, want) {
		t.Errorf("Genres() = %v, want %v", got, want)
	}
	pairs := map[Genre][]Subgenre{
		GenreAdventure:  {SubgenreAdventureRounds},
		GenreNPC:        {SubgenreNPCLive, SubgenreNPCRounds},
		GenreDiscussion: {SubgenreDiscussionLive},
	}
	for g, want := range pairs {
		if got := g.Subgenres(); !slices.Equal(got, want) {
			t.Errorf("%s.Subgenres() = %v, want %v", g, got, want)
		}
		for _, sub := range want {
			if sub.Genre() != g {
				t.Errorf("%s.Genre() = %q, want %q", sub, sub.Genre(), g)
			}
		}
	}
	modes := map[Subgenre]Mode{
		SubgenreAdventureRounds: ModeRounds, SubgenreNPCLive: ModeLive,
		SubgenreNPCRounds: ModeRounds, SubgenreDiscussionLive: ModeLive,
	}
	for sub, want := range modes {
		if sub.Mode() != want {
			t.Errorf("%s.Mode() = %q, want %q", sub, sub.Mode(), want)
		}
	}
}

func TestGenresFollowFromToggles(t *testing.T) {
	everything := Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true}
	cases := []struct {
		name      string
		toggles   Toggles
		genres    []Genre
		subgenres []Subgenre
	}{
		{"nothing on", Toggles{}, []Genre{GenreAdventure, GenreNPC}, []Subgenre{SubgenreAdventureRounds, SubgenreNPCRounds}},
		{"images only", Toggles{Images: true}, []Genre{GenreAdventure, GenreNPC}, []Subgenre{SubgenreAdventureRounds, SubgenreNPCRounds}},
		// The live genre's picture comes from the gallery, so it needs no image model.
		{"live only", Toggles{AudioLive: true}, []Genre{GenreAdventure, GenreNPC, GenreDiscussion},
			[]Subgenre{SubgenreAdventureRounds, SubgenreNPCLive, SubgenreNPCRounds, SubgenreDiscussionLive}},
		{"everything", everything, []Genre{GenreAdventure, GenreNPC, GenreDiscussion},
			[]Subgenre{SubgenreAdventureRounds, SubgenreNPCLive, SubgenreNPCRounds, SubgenreDiscussionLive}},
	}
	for _, c := range cases {
		if got := PlayableGenres(c.toggles); !slices.Equal(got, c.genres) {
			t.Errorf("%s: genres %v, want %v", c.name, got, c.genres)
		}
		if got := PlayableSubgenres(c.toggles); !slices.Equal(got, c.subgenres) {
			t.Errorf("%s: subgenres %v, want %v", c.name, got, c.subgenres)
		}
	}
}

// The player picks live or rounds where both fit the offers; where one
// fits, it plays, whatever was asked, as a mode is a wish and never a lock.
func TestPickSubgenre(t *testing.T) {
	live := MockOffer()
	plain := Offer{Platform: PlatformMock, ID: "plain", Caveats: Caveats{
		Allow:   map[Tier]Toggles{TierMedium: {Images: true}},
		Default: Choice{Tier: TierMedium, Toggles: Toggles{Images: true}},
	}}
	cases := []struct {
		name    string
		genre   Genre
		mode    Mode
		offers  []Offer
		want    Subgenre
		problem Problem
	}{
		{"both fit, none asked", GenreNPC, "", []Offer{live}, "", ProblemChooseMode},
		{"both fit, live asked", GenreNPC, ModeLive, []Offer{live}, SubgenreNPCLive, ""},
		{"both fit, rounds asked", GenreNPC, ModeRounds, []Offer{live}, SubgenreNPCRounds, ""},
		{"only rounds fits", GenreNPC, "", []Offer{plain}, SubgenreNPCRounds, ""},
		{"live asked, only rounds fits", GenreNPC, ModeLive, []Offer{plain}, SubgenreNPCRounds, ""},
		{"one half", GenreAdventure, "", []Offer{live}, SubgenreAdventureRounds, ""},
		{"the missing half asked", GenreAdventure, ModeLive, []Offer{live}, SubgenreAdventureRounds, ""},
		{"no half fits", GenreDiscussion, "", []Offer{plain}, "", ProblemGenre},
		{"either offer", GenreNPC, "", []Offer{plain, live}, "", ProblemChooseMode},
	}
	for _, c := range cases {
		got, err := PickSubgenre(c.genre, c.mode, c.offers)
		if c.problem != "" {
			if err == nil || RefusalOf(err).Problem != c.problem {
				t.Errorf("%s: err %v, want problem %q", c.name, err, c.problem)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	if _, err := PickSubgenre("chess", "", []Offer{live}); err == nil {
		t.Error("an unknown genre was picked from")
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

// An editor offers a genre's own fields, the same for both halves: an NPC
// is one character's voice, with no status, and the image style its cover is
// drawn in; an Adventure has all of them, and its narrator a voice.
func TestGenresNameTheirFields(t *testing.T) {
	fields := map[Genre][]GameField{}
	for _, g := range GenreList() {
		fields[g.ID] = g.Fields
	}
	npc := fields[GenreNPC]
	if !slices.Contains(npc, FieldVoiceStyle) || !slices.Contains(npc, FieldImageStyle) || slices.Contains(npc, FieldStatusFields) {
		t.Errorf("npc fields: %v", npc)
	}
	adventure := fields[GenreAdventure]
	for _, want := range []GameField{FieldScenario, FieldGameStart, FieldImageStyle, FieldStatusFields, FieldVoiceStyle} {
		if !slices.Contains(adventure, want) {
			t.Errorf("adventure lacks %q: %v", want, adventure)
		}
	}
}

// GET genres lists each genre with its halves, the toggles each needs and
// its prompts; a genre needs what all its halves need.
func TestGenreListShowsThePairs(t *testing.T) {
	list := GenreList()
	var npc *GenreInfo
	for i := range list {
		if list[i].ID == GenreNPC {
			npc = &list[i]
		}
	}
	if npc == nil {
		t.Fatalf("GenreList has no npc: %v", list)
	}
	if len(npc.Needs) != 0 {
		t.Errorf("npc needs %v, want none: npc-rounds runs without a toggle", npc.Needs)
	}
	if len(npc.Subgenres) != 2 {
		t.Fatalf("npc subgenres: %+v", npc.Subgenres)
	}
	liveHalf, rounds := npc.Subgenres[0], npc.Subgenres[1]
	if liveHalf.ID != SubgenreNPCLive || liveHalf.Mode != ModeLive || !slices.Equal(liveHalf.Needs, []Toggle{ToggleAudioLive}) {
		t.Errorf("live half: %+v", liveHalf)
	}
	if rounds.ID != SubgenreNPCRounds || rounds.Mode != ModeRounds || rounds.Needs == nil || len(rounds.Needs) != 0 {
		t.Errorf("rounds half: %+v", rounds)
	}
	if len(liveHalf.Prompts) == 0 || len(rounds.Prompts) == 0 {
		t.Error("a half lists no prompts")
	}
	for _, info := range list {
		if info.ID == GenreDiscussion && !slices.Equal(info.Needs, []Toggle{ToggleAudioLive}) {
			t.Errorf("discussion needs %v, want audio-live", info.Needs)
		}
	}
}

// Each half names the voices it speaks in, its voice slots, and what a game
// may choose for each: a built-in voice of a gender, or any the cast finds
// fitting; and a voice of its own only where the graph makes one. A genre's
// slots are its halves', a voice of one name being one slot: an NPC's
// character may have a voice of its own, which NPC-Live speaks in. An editor
// offers exactly these.
func TestGenresNameTheirVoiceSlots(t *testing.T) {
	builtin := []VoiceChoice{VoiceMale, VoiceFemale, VoiceAny}
	own := []VoiceChoice{VoiceMale, VoiceFemale, VoiceAny, VoiceCustom}
	want := map[Genre][]VoiceSlot{
		GenreAdventure:  {{Name: "narrator", Choices: builtin}},
		GenreNPC:        {{Name: "character", Choices: own}},
		GenreDiscussion: {{Name: "a", Choices: builtin}, {Name: "b", Choices: builtin}},
	}
	for _, info := range GenreList() {
		if !reflect.DeepEqual(info.VoiceSlots, want[info.ID]) {
			t.Errorf("%s voice slots = %+v, want %+v", info.ID, info.VoiceSlots, want[info.ID])
		}
	}
	halves := map[Subgenre][]VoiceSlot{
		SubgenreNPCLive:   {{Name: "character", Choices: own}},
		SubgenreNPCRounds: {{Name: "character", Choices: builtin}},
	}
	for half, slots := range halves {
		if got := half.VoiceSlots(); !reflect.DeepEqual(got, slots) {
			t.Errorf("%s voice slots = %+v, want %+v", half, got, slots)
		}
	}
}

// A spec names its game's genre and the player's wish; the launch picks
// the half. A session stored before genres came in pairs named its half as
// its genre, and resumes as that half.
func TestSpecResolvesItsHalf(t *testing.T) {
	cases := []struct {
		spec  SessionSpec
		genre Genre
		half  Subgenre
	}{
		{SessionSpec{Genre: GenreNPC, Mode: ModeRounds}, GenreNPC, SubgenreNPCRounds},
		{SessionSpec{Subgenre: SubgenreNPCLive}, GenreNPC, SubgenreNPCLive},
		{SessionSpec{Genre: "npc-live"}, GenreNPC, SubgenreNPCLive},
		{SessionSpec{Genre: GenreAdventure}, GenreAdventure, SubgenreAdventureRounds},
		{SessionSpec{Genre: GenreDiscussion}, GenreDiscussion, SubgenreDiscussionLive},
	}
	for _, c := range cases {
		got, err := c.spec.resolved([]Offer{MockOffer()})
		if err != nil || got.Genre != c.genre || got.Subgenre != c.half {
			t.Errorf("%+v resolved to %s/%s, %v; want %s/%s", c.spec, got.Genre, got.Subgenre, err, c.genre, c.half)
		}
	}
	_, err := SessionSpec{Genre: GenreNPC}.resolved([]Offer{MockOffer()})
	if err == nil || RefusalOf(err).Problem != ProblemChooseMode {
		t.Errorf("both halves fitting and no wish: %v, want choose_mode", err)
	}
	issues := Validate(SessionSpec{Genre: GenreAdventure, Subgenre: SubgenreNPCLive}, []Offer{MockOffer()})
	if !hasError(issues) {
		t.Error("a half of another genre passed")
	}
}
