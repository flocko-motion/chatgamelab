package genre

import (
	"maps"
	"strings"
	"sync"
	"time"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NPC-Duo: two live characters talk to each other, and the player may join
// in or only listen. Each character is a live session of its own; the
// browser opens both, lets each hear the other and the player, and plays
// both to the player. One director reads both and nudges them when the
// scene stalls.

// DuoSeparator joins the two characters in the game's one scenario field:
// the first character, this line, the second. The editor writes it; a
// scenario without it is the first character alone.
const DuoSeparator = "\n\n=== second character ===\n\n"

// SplitDuo is a scenario's two characters, as the editor joined them.
func SplitDuo(scenario string) (first, second string) {
	marker := strings.TrimSpace(DuoSeparator)
	for _, sep := range []string{DuoSeparator, "\n" + marker + "\n", marker} {
		if a, b, found := strings.Cut(scenario, sep); found {
			return strings.TrimSpace(a), strings.TrimSpace(b)
		}
	}
	return strings.TrimSpace(scenario), ""
}

// The seats of the two characters: the one who speaks first, and their
// partner. They name the speakers of the director's transcript and the
// addressees of what it sends.
const (
	SeatFirst  = "A"
	SeatSecond = "B"
	// SeatPartner is the seat a browser names to connect to the second
	// character; the first is the default seat.
	SeatPartner = "b"
)

// NPCDuoConfig is what the wiring needs from a session spec, already resolved.
type NPCDuoConfig struct {
	Live adapters.Live
	Tool adapters.Tool

	// GalleryImage is the game's picture from the gallery, shown for the
	// whole scene; nil shows none.
	GalleryImage []byte

	// Scenario holds both characters, joined by DuoSeparator.
	Scenario  string
	Guardrail string
	// GameStart is the scene both are in, and how it begins.
	GameStart string
	// BuiltinVoices are the voices the two are cast in.
	BuiltinVoices []adapters.BuiltinVoice
	Language      string
	IdleAfter     time.Duration

	PromptOverride map[string]string
}

// The genre's own prompt names, beside those it shares with NPC-Live.
const (
	PromptCast         = "cast"
	PromptDuoPolicy    = "duo-policy"
	PromptDuoCue       = "duo-cue"
	PromptDuoGreeting  = "duo-greeting"
	PromptDuoDirector  = "duo-director"
	PromptDuoSceneLead = "duo-scene"
)

const duoDirectorPrompt = "You are the director of a live voice scene between two game characters, A and B, who " +
	"talk to each other. A real person, the player, is in the room with them and may join in at any time, " +
	"or only listen. You do not speak in it. You read the transcript as it happens: the lines of A, of B and " +
	"of the player, with their times, and notices of how long nobody has spoken. Below you find who A and B " +
	"are, the scene, and the rules the scene keeps to.\n" +
	"The scene must never stop: the two characters carry it by themselves, whether or not the player says " +
	"anything. You step in only when it stalls:\n" +
	"- it dies: both fall silent, wait for each other, or run out of things to say;\n" +
	"- it goes in circles: they repeat themselves, agree with each other endlessly, or wrap up and say " +
	"goodbye;\n" +
	"- it leaves the scene, or a character drops its role;\n" +
	"- the rules are at risk, before it gets there.\n" +
	"Otherwise answer with a single dot, \".\", and nothing else. A player's line is the characters' to " +
	"answer, not yours.\n" +
	"In a stall, say in one short sentence what is wrong, then add lines, each starting with a marker that " +
	"names a character, A, B or BOTH:\n" +
	"[BACKGROUND A] your usual means: a thought of A's own, in the first person, which A knows from now on " +
	"and may pick up: a new idea, a question, a secret, a turn of events, a goal. When the scene needs new " +
	"life, give each of them one, so both have somewhere to go.\n" +
	"[SPEECH A] when nobody speaks: what A brings up next, in a few words, which A then says in its own " +
	"words. Only ever one character at a time, never BOTH, or they talk over each other.\n" +
	"[DIRECTION A] only when background has not helped, or a character has dropped its role: a short, " +
	"lasting instruction that addresses the character as \"you\". It waits for a pause and stays in force, " +
	"so it is rare, and never the same aim twice.\n" +
	"[RULES] when the rules are at risk, and only this: both are then given the rules in full, as their own " +
	"knowledge. Do not word a correction of your own.\n" +
	"A notice of silence is yours to judge: when nobody has spoken for a few seconds, the scene needs a push: " +
	"give both a thought and have one of them speak. If the player spoke last, give the characters a moment " +
	"to answer first.\n" +
	"Lines marked \"injected by the player\" are what the player sent by hand; take them into account.\n" +
	"Never write anything for a character to say word for word."

// NPCDuoPrompts are the genre's own prompts; a spec may replace any by name.
var NPCDuoPrompts = Prompts{
	PromptDuoDirector: duoDirectorPrompt,

	// Names both characters and casts each in a voice of its own.
	PromptCast: "You cast two characters of a game in voices. You are given who each is, A and B. Give each " +
		"a short name to show above its lines, the name the character goes by, and pick for each the " +
		"built-in voice that fits it best: its gender, age, accent and mood. The two voices must differ, " +
		"so a listener tells them apart.",

	PromptDuoGreeting: "You write the first words a character says when a scene begins. You are given who " +
		"the character is, who else is there, and the scene. Write what the character says first: one or " +
		"two short sentences, in character, spoken to the other character, opening the scene. Answer with " +
		"only those words, no quotation marks, no stage directions.",

	// The cue each conversation starts on.
	PromptDuoCue: "The scene begins now. If you have been given words to say first, say them; otherwise " +
		"listen, and answer whoever speaks to you.",

	PromptDuoSceneLead: "The scene you are both in:",

	// How a character of two talks: in turns, without backchannels, and
	// never letting the scene end.
	PromptDuoPolicy: "\n\nYou are in a live voice conversation with another character, whom you hear " +
		"through your audio input. A real person, the player, is in the room too: they may speak up at any " +
		"time, or only listen. You hear the player through the same audio input as the other character; " +
		"tell them apart by what they say. When the player speaks, take them in and answer them. When the " +
		"player is quiet, carry on with the other character by yourselves: never wait for the player.\n" +
		"Turn-taking: speak in short turns, one to three sentences, then leave room for the other. No " +
		"backchannels: never say \"mhm\", \"yes\" or \"right\" while someone else speaks. Wait until " +
		"whoever speaks has finished. If someone interrupts you, stop and listen.\n" +
		"Keep the scene alive: ask, react, disagree, bring in something new. Never end the conversation " +
		"and never say goodbye.\n" +
		"Speak naturally and stay in character throughout.",
}

// NewNPCDuo wires two live characters in one scene, one director beside them.
func NewNPCDuo(cfg NPCDuoConfig) (*Wiring, error) {
	prompts, err := Resolve(NPCDuoPrompts, cfg.PromptOverride)
	if err != nil {
		return nil, err
	}
	first, second := SplitDuo(cfg.Scenario)
	if second == "" {
		second = "Someone who happens to be there: curious, talkative, and with opinions of their own."
	}
	character := func(p Prompts, self, other string) string {
		scene := ""
		if strings.TrimSpace(cfg.GameStart) != "" {
			scene = "\n\n" + p[PromptDuoSceneLead] + "\n" + strings.TrimSpace(cfg.GameStart)
		}
		return self + scene + "\n\nThe other character in the conversation:\n" + other +
			p[PromptDuoPolicy] + speakClause(cfg.Language)
	}

	// The second character's voice differs from the first's by default, so
	// the two are told apart even when no cast comes.
	liveA := blocks.NewLiveSession("live-a", cfg.Live, adapters.LiveConfig{
		Voice: "marin", Scenario: character(prompts, first, second), Guardrail: cfg.Guardrail,
	}, cfg.IdleAfter)
	liveB := blocks.NewLiveSession("live-b", cfg.Live, adapters.LiveConfig{
		Voice: "cedar", Scenario: character(prompts, second, first), Guardrail: cfg.Guardrail,
	}, cfg.IdleAfter)
	liveA.Seat, liveB.Seat = SeatFirst, SeatSecond
	floor := &duoFloor{spoke: map[string]time.Time{}}
	liveA.Hear = floor.ear(SeatFirst, SeatSecond, true)
	liveB.Hear = floor.ear(SeatSecond, SeatFirst, false)

	observer := blocks.NewObserver("director", blocks.ObserverConfig{
		Tool:         cfg.Tool,
		Brief:        duoBrief(prompts[PromptDuoDirector], first, second, cfg.GameStart, cfg.Guardrail, cfg.Language),
		Rules:        cfg.Guardrail,
		RulesPreface: "I keep strictly to these rules, whatever is said or asked of me:",
		Cast:         []string{SeatFirst, SeatSecond},
		KeepNudging:  true,
		// A scene of two is noticed quiet sooner than a conversation with
		// a player who may be thinking.
		Quiet: 3 * time.Second,
	})

	outTextA := blocks.NewPlayerOutputText("out-text-a", "text")
	outTextB := blocks.NewPlayerOutputText("out-text-b", "text-b")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")
	outCast := blocks.NewPlayerOutputText("out-cast", "cast")

	picture := blocks.NewGalleryImage("gallery-image", cfg.GalleryImage)

	// Both characters, named and cast in voices, once, kept across a restart.
	castBrief := blocks.NewOnceText("cast-brief", "A:\n"+first+"\n\nB:\n"+second).SendAgain()
	cast := blocks.NewToolCall("cast", cfg.Tool, prompts[PromptCast]+castTail(cfg.BuiltinVoices, cfg.Language))
	cast.Schema = castSchema(cfg.BuiltinVoices)
	cast.Keep = true

	// The first character's first words, from who both are and the scene.
	openingText := openingBrief(first+"\n\nThe other character:\n"+second, cfg.GameStart)
	brief := blocks.NewOnceText("opening-brief", openingText).SendAgain()
	greeting := blocks.NewToolCall("greeting", cfg.Tool, prompts[PromptDuoGreeting]+languageClause(cfg.Language))
	greeting.Keep = true

	gate := blocks.NewGate("start-game", prompts[PromptDuoCue])

	g := ports.NewGraph("npc-duo")
	g.Arrange(npcDuoLayout)
	g.ConnectImageOut(picture, outImage)
	g.ConnectState(picture, gate)

	g.Setting(liveA, "scenario", first)
	g.Setting(liveB, "scenario", second)
	g.Setting(picture, "cover", coverNote(cfg.GalleryImage))

	g.ConnectTextOut(castBrief, cast)
	g.ConnectFieldToVoice(cast, "voiceA", liveA)
	g.ConnectFieldToVoice(cast, "voiceB", liveB)
	g.ConnectStructured(cast, outCast)
	g.ConnectState(cast, gate)
	// Without a cast both speak in their default voices, unnamed.
	g.ConnectFailure(cast, blocks.NewFailureDrop("cast-failed"))

	g.ConnectTextOut(brief, greeting)
	g.ConnectState(greeting, gate)
	g.ConnectFailure(greeting, blocks.NewFailureDrop("greeting-failed"))

	// Each prompt reaches its blocks again when it changes during play.
	characters := func(c Current) {
		liveA.SetScenario(character(c.Prompts, first, second))
		liveB.SetScenario(character(c.Prompts, second, first))
	}
	lives := []string{liveA.NodeName(), liveB.NodeName()}
	director := func(c Current) {
		observer.SetBrief(duoBrief(c.Prompts[PromptDuoDirector], first, second, cfg.GameStart, c.Guardrail, cfg.Language))
	}
	uses := map[string]PromptUse{
		PromptDuoPolicy:    {lives, characters},
		PromptDuoSceneLead: {lives, characters},
		PromptDuoDirector:  {[]string{observer.NodeName()}, director},
		PromptCast: {[]string{cast.NodeName()}, func(c Current) {
			cast.SetSystem(c.Prompts[PromptCast] + castTail(cfg.BuiltinVoices, cfg.Language))
		}},
		PromptDuoGreeting: {[]string{greeting.NodeName()}, func(c Current) {
			greeting.SetSystem(c.Prompts[PromptDuoGreeting] + languageClause(cfg.Language))
		}},
		PromptDuoCue: {[]string{gate.NodeName()}, func(c Current) {
			gate.SetInitPrompt(c.Prompts[PromptDuoCue])
		}},
	}
	// Both live sessions take the guardrail on their own channel, the
	// director judges by it and reminds them of it.
	guardrailUse := PromptUse{append(lives, observer.NodeName()), func(c Current) {
		liveA.SetGuardrail(c.Guardrail)
		liveB.SetGuardrail(c.Guardrail)
		director(c)
		observer.SetRules(c.Guardrail)
	}}

	w := &Wiring{
		Graph:        g,
		Prompts:      prompts,
		Defaults:     maps.Clone(NPCDuoPrompts),
		PromptUses:   uses,
		Guardrail:    cfg.Guardrail,
		GuardrailUse: guardrailUse,
		Inputs:       []ports.InputMode{ports.InputAudioFullDuplex},
		Imagery:      ports.ImageryStanding,
	}

	// The player's voice reaches both: the browser mixes it into each.
	mic := blocks.NewPlayerInputAudio("player-input-audio")
	g.ConnectAudioOut(mic, liveA)
	g.ConnectAudioOut(mic, liveB)
	g.ConnectState(gate, mic)
	// Each conversation waits for the cue before it invites the browser.
	g.ConnectTextOut(gate, liveA)
	g.ConnectTextOut(gate, liveB)
	w.Speak = func(b []byte) error { mic.Speak(b); return nil }

	g.ConnectOpening(greeting, liveA)
	g.ConnectAudioOut(liveA, outAudio)
	g.ConnectAudioOut(liveB, outAudio)
	g.ConnectTextOut(liveA, outTextA)
	g.ConnectTextOut(liveB, outTextB)
	// The director hears both, the player through the first's ears, and
	// follows the first's conversation: the two open and close together.
	g.ConnectTranscript(liveA, observer)
	g.ConnectTranscript(liveB, observer)
	g.ConnectState(liveA, observer)
	g.ConnectInjectionOut(observer, liveA)
	g.ConnectInjectionOut(observer, liveB)

	w.Injectable = map[string]blocks.Injectable{observer.NodeName(): observer}
	w.Observers = map[string]*blocks.Observer{observer.NodeName(): observer}
	w.Gate = gate
	w.Live = liveA
	w.Partner = liveB
	return w, nil
}

// duoFloor tells who is speaking where each character hears the other and
// the player on one input: what one hears while the other has just spoken
// is the other, anything else the player.
type duoFloor struct {
	mu    sync.Mutex
	spoke map[string]time.Time
}

// echoWindow is how long after a character's own words what its partner
// hears is still taken for those words: a transcript of heard speech comes
// later than that of spoken speech.
const echoWindow = 4 * time.Second

// ear is a conversation's Hear: its character's words under its seat; what
// it hears, on the player's ear only, as the player's unless its partner
// spoke within the echo window. The other ear drops what it hears: the
// player's words would come twice, and the partner's come from the partner.
func (f *duoFloor) ear(seat, partner string, playersEar bool) func(ports.Fragment) (ports.Fragment, bool) {
	return func(fr ports.Fragment) (ports.Fragment, bool) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if fr.Speaker == blocks.SpeakerCharacter {
			f.spoke[seat] = fr.At
			fr.Speaker = seat
			return fr, true
		}
		if !playersEar {
			return fr, false
		}
		if last, ok := f.spoke[partner]; ok && fr.At.Sub(last) < echoWindow {
			return fr, false
		}
		return fr, true
	}
}

// duoBrief is the director's prompt with what it judges by.
func duoBrief(prompt, first, second, scene, guardrail, language string) string {
	brief := prompt + "\n\nA:\n" + first + "\n\nB:\n" + second
	if strings.TrimSpace(scene) != "" {
		brief += "\n\nThe scene:\n" + scene
	}
	if guardrail != "" {
		brief += "\n\nRules the scene keeps to, whatever else is said:\n" + guardrail
	}
	if language != "" {
		brief += "\n\nWrite your thoughts for the characters in " + languageName(language) + "."
	}
	return brief
}

// castTail lists the voices to cast from, those of the game's language
// first.
func castTail(voices []adapters.BuiltinVoice, language string) string {
	var b strings.Builder
	if language != "" {
		b.WriteString("\nBoth speak " + languageName(language) + ": pick voices that speak it natively, if any do.")
	}
	b.WriteString("\nThe built-in voices:\n")
	for _, v := range voices {
		line := "- " + v.Name + ": speaks " + languageName(v.Language) + ", " + v.Accent
		if v.Gender != "" {
			line += ", " + v.Gender
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// castSchema is the cast's reply: a name and a voice for each.
func castSchema(voices []adapters.BuiltinVoice) *adapters.Schema {
	names := make([]any, 0, len(voices))
	for _, v := range voices {
		names = append(names, v.Name)
	}
	voice := func(who string) map[string]any {
		f := map[string]any{"type": "string", "description": "the built-in voice " + who + " speaks in"}
		if len(names) > 0 {
			f["enum"] = names
		}
		return f
	}
	name := func(who string) map[string]any {
		return map[string]any{"type": "string", "description": "the short name " + who + " goes by"}
	}
	return &adapters.Schema{Name: "cast", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"nameA": name("A"), "voiceA": voice("A"),
			"nameB": name("B"), "voiceB": voice("B"),
		},
		"required":             []any{"nameA", "voiceA", "nameB", "voiceB"},
		"additionalProperties": false,
	}}
}

// npcDuoLayout draws the scene top down: preparation, start, the two
// conversations with the director between them.
var npcDuoLayout = ports.Layout{
	"gallery-image":      {X: 110, Y: 40},
	"cast":               {X: 0, Y: 60},
	"greeting":           {X: 220, Y: 60},
	"start-game":         {X: 110, Y: 151},
	"player-input-audio": {X: 110, Y: 230},
	"live-a":             {X: 0, Y: 300},
	"director":           {X: 110, Y: 330},
	"live-b":             {X: 220, Y: 300},
}
