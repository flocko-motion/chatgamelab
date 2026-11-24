package engine

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/genre"
)

// The catalogue's parts, re-exported so a caller can read them without importing internal/.
type (
	Toggle     = adapters.Toggle
	Toggles    = adapters.Toggles
	ModelType  = adapters.ModelType
	Model      = adapters.Model
	ModelSet   = adapters.ModelSet
	TierModels = adapters.TierModels
	Medium     = adapters.Medium
	ModelFacts = adapters.ModelFacts
	ModelTable = adapters.ModelTable
	Price      = adapters.Price
)

const (
	ToggleImages    = adapters.ToggleImages
	ToggleAudioOut  = adapters.ToggleAudioOut
	ToggleAudioIn   = adapters.ToggleAudioIn
	ToggleAudioLive = adapters.ToggleAudioLive

	ModelTooling = adapters.ModelTooling
	ModelStory   = adapters.ModelStory
	ModelImage   = adapters.ModelImage
	ModelLive    = adapters.ModelLive
	ModelSpeech  = adapters.ModelSpeech
	ModelHearing = adapters.ModelHearing
)

// PlatformInfo is one AI platform the engine can run: what a person choosing a
// platform and a tier needs to know. Genres follow from Toggles. Models says
// what each model is: what it takes and makes, and its price, dated, with its
// source.
type PlatformInfo struct {
	ID       Platform     `json:"id"`
	Name     string       `json:"name"`
	UserKeys bool         `json:"userKeys"`
	Toggles  Toggles      `json:"toggles"`
	Genres   []Genre      `json:"genres"`
	Tiers    []TierModels `json:"tiers"`
	Models   ModelTable   `json:"models"`
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
			Models:   p.Facts,
		})
	}
	return out
}

// AcceptsKeys reports whether the engine runs a platform on user API keys.
func AcceptsKeys(id Platform) bool {
	return slices.ContainsFunc(Platforms(), func(p PlatformInfo) bool { return p.ID == id && p.UserKeys })
}

// GenreNamed is the genre a name stands for: a half's name or a former
// genre's, as a game file may hold it, stands for its genre ("npc-live" an
// NPC, "npc-duo" a Discussion). Any other name is taken as it is.
func GenreNamed(name string) Genre {
	switch name {
	case "npc-live":
		return GenreNPC
	case "npc-duo":
		return GenreDiscussion
	}
	return Genre(name)
}

// Genres is every genre a game can be.
func Genres() []Genre { return []Genre{GenreAdventure, GenreNPC, GenreDiscussion} }

// Subgenres is every half of every genre the engine runs.
func Subgenres() []Subgenre {
	return []Subgenre{SubgenreAdventureRounds, SubgenreNPCLive, SubgenreNPCRounds, SubgenreDiscussionLive}
}

// Subgenres are the genre's halves the engine runs, live first.
func (g Genre) Subgenres() []Subgenre {
	var out []Subgenre
	for _, s := range Subgenres() {
		if s.Genre() == g {
			out = append(out, s)
		}
	}
	return out
}

// Subgenre is the genre's half of a mode, if the engine runs it.
func (g Genre) Subgenre(mode Mode) (Subgenre, bool) {
	s := Subgenre(string(g) + "-" + string(mode))
	return s, slices.Contains(g.Subgenres(), s)
}

// Genre is the genre the half belongs to.
func (s Subgenre) Genre() Genre {
	name, _ := s.split()
	return Genre(name)
}

// Mode is which half it is.
func (s Subgenre) Mode() Mode {
	_, mode := s.split()
	return mode
}

func (s Subgenre) split() (string, Mode) {
	for _, mode := range []Mode{ModeLive, ModeRounds} {
		if name, ok := strings.CutSuffix(string(s), "-"+string(mode)); ok {
			return name, mode
		}
	}
	return string(s), ""
}

// Needs is the toggles a half cannot run without.
func (s Subgenre) Needs() []Toggle {
	if s.Mode() == ModeLive {
		return []Toggle{ToggleAudioLive}
	}
	return nil
}

// Needs is the toggles a genre cannot run without at all: those every one
// of its halves needs.
func (g Genre) Needs() []Toggle {
	var out []Toggle
	halves := g.Subgenres()
	for _, toggle := range adapters.AllToggles() {
		all := len(halves) > 0
		for _, s := range halves {
			all = all && slices.Contains(s.Needs(), toggle)
		}
		if all {
			out = append(out, toggle)
		}
	}
	return out
}

// GameField is a field of a game that a genre reads, by the name a client
// edits it under.
type GameField string

const (
	FieldScenario     GameField = "scenario"
	FieldGameStart    GameField = "gameStart"
	FieldImageStyle   GameField = "imageStyle"
	FieldVoiceStyle   GameField = "voiceStyle"
	FieldStatusFields GameField = "statusFields"
)

// Fields is the game's fields the genre reads, the same for both halves, so
// an editor offers those and no others: an Adventure tells a world in
// pictures and status, and may be narrated; an NPC is one character's voice.
// Every genre has the image style, as every game's cover is drawn in it.
func (g Genre) Fields() []GameField {
	switch g {
	case GenreNPC:
		return []GameField{FieldScenario, FieldGameStart, FieldImageStyle, FieldVoiceStyle}
	case GenreDiscussion:
		// Both characters share the scenario field; their voices are cast
		// from who they are.
		return []GameField{FieldScenario, FieldGameStart, FieldImageStyle}
	}
	return []GameField{FieldScenario, FieldGameStart, FieldImageStyle, FieldStatusFields, FieldVoiceStyle}
}

// GenrePrompt is one of a genre's own prompts as it is written: a template
// whose fields are filled for each session, which a spec's override may use
// too.
type GenrePrompt struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

// Prompts are the half's own prompts, by name, sorted: what a session runs
// on unless its spec replaces one.
func (s Subgenre) Prompts() []GenrePrompt {
	defaults := s.defaultPrompts()
	out := []GenrePrompt{}
	for _, name := range slices.Sorted(maps.Keys(defaults)) {
		out = append(out, GenrePrompt{Name: name, Text: defaults[name]})
	}
	return out
}

func (s Subgenre) defaultPrompts() genre.Prompts {
	switch s {
	case SubgenreAdventureRounds:
		return genre.AdventurePrompts
	case SubgenreNPCLive:
		return genre.NPCLivePrompts
	case SubgenreNPCRounds:
		return genre.NPCRoundsPrompts
	case SubgenreDiscussionLive:
		return genre.NPCDuoPrompts
	}
	return nil
}

// VoiceChoice is what a game chooses for one of a genre's voices.
type VoiceChoice = genre.VoiceChoice

const (
	VoiceMale   = genre.VoiceMale
	VoiceFemale = genre.VoiceFemale
	VoiceAny    = genre.VoiceAny
	VoiceCustom = genre.VoiceCustom
)

// VoiceSlot is one voice a genre speaks in, by the name a game chooses for
// it, and the choices it has.
type VoiceSlot struct {
	Name    string        `json:"name"`
	Choices []VoiceChoice `json:"choices"`
}

// VoiceSlots are the voices the half speaks in, read off its wiring: every
// voice is cast from the built-in voices, so a game may ask for a gender or
// any; a voice of its own only where a block that makes one feeds it.
func (s Subgenre) VoiceSlots() []VoiceSlot {
	session := &Session{usage: newUsageLedger()}
	w, err := session.wire(SessionSpec{Subgenre: s}, PlatformOpenAI)
	if err != nil {
		return []VoiceSlot{}
	}
	out := []VoiceSlot{}
	for _, v := range w.Graph.Voices() {
		choices := []VoiceChoice{VoiceMale, VoiceFemale, VoiceAny}
		if v.Maker {
			choices = append(choices, VoiceCustom)
		}
		out = append(out, VoiceSlot{Name: v.Name, Choices: choices})
	}
	return out
}

// VoiceSlots are the voices the genre speaks in, those of both halves: a
// voice both speak in, by one name, is one slot, which a game chooses for
// once, with every choice either half offers.
func (g Genre) VoiceSlots() []VoiceSlot {
	out := []VoiceSlot{}
	for _, half := range g.Subgenres() {
		for _, slot := range half.VoiceSlots() {
			i := slices.IndexFunc(out, func(v VoiceSlot) bool { return v.Name == slot.Name })
			if i < 0 {
				out = append(out, slot)
				continue
			}
			for _, c := range slot.Choices {
				if !slices.Contains(out[i].Choices, c) {
					out[i].Choices = append(out[i].Choices, c)
				}
			}
		}
	}
	return out
}

// MakesVoices reports whether a half of the genre makes a voice of its own
// for a game, kept per key and voice style: whether a voice of it offers
// custom.
func (g Genre) MakesVoices() bool {
	return slices.ContainsFunc(g.VoiceSlots(), func(v VoiceSlot) bool { return slices.Contains(v.Choices, VoiceCustom) })
}

// PlayableSubgenres is every half whose needs the toggles meet.
func PlayableSubgenres(toggles Toggles) []Subgenre {
	var out []Subgenre
	for _, s := range Subgenres() {
		if missing(s, toggles) == nil {
			out = append(out, s)
		}
	}
	return out
}

// PlayableGenres is every genre with a half the toggles run: a game plays
// where one of its halves does.
func PlayableGenres(toggles Toggles) []Genre {
	playable := PlayableSubgenres(toggles)
	var out []Genre
	for _, g := range Genres() {
		if slices.ContainsFunc(g.Subgenres(), func(s Subgenre) bool { return slices.Contains(playable, s) }) {
			out = append(out, g)
		}
	}
	return out
}

// PickSubgenre is the half of a genre a session plays on the offers: the one
// that fits, where only one does; the mode asked for, where both do. A mode
// is a wish, never a lock, so a mode that does not fit gives way to the half
// that does. Both fitting and no mode asked is refused with
// ProblemChooseMode, for the player to choose; none fitting with
// ProblemGenre.
func PickSubgenre(g Genre, mode Mode, offers []Offer) (Subgenre, error) {
	halves := g.Subgenres()
	if len(halves) == 0 {
		return "", fmt.Errorf("unknown genre %q", g)
	}
	sound := soundOffers(offers)
	var fitting []Subgenre
	for _, s := range halves {
		if len(runnableOffers(s, offers, sound)) > 0 {
			fitting = append(fitting, s)
		}
	}
	switch {
	case len(fitting) == 0:
		return "", Refuse(ProblemGenre, fmt.Sprintf("no offer can run %s", g))
	case len(fitting) == 1:
		return fitting[0], nil
	}
	if s, ok := g.Subgenre(mode); ok && slices.Contains(fitting, s) {
		return s, nil
	}
	return "", Refuse(ProblemChooseMode, fmt.Sprintf("%s plays live or in rounds: choose one", g))
}

func missing(s Subgenre, toggles Toggles) []Toggle {
	var out []Toggle
	for _, need := range s.Needs() {
		if !toggles.Has(need) {
			out = append(out, need)
		}
	}
	return out
}
