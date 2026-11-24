package genre

import (
	"fmt"
	"maps"
	"strings"
	"time"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NPCLiveConfig is what the wiring needs from a session spec, already resolved.
type NPCLiveConfig struct {
	Live adapters.Live
	// Tool writes the character's first words before the game opens.
	Tool adapters.Tool

	// GalleryImage is the game's picture from the gallery, shown before the
	// conversation; nil shows none. Nothing is painted, so the conversation
	// starts at once.
	GalleryImage []byte

	// Scenario and Guardrail stay apart: nothing a designer writes may reach the guardrail.
	Scenario  string
	Guardrail string
	Voice     string
	// VoiceStyle is how the character sounds, in the designer's words; it
	// goes with the scenario, labelled as the designer's, and a voice is made
	// from it on the session's key.
	VoiceStyle string
	// Voices makes and keeps the game's voice on the settled key; nil, or
	// empty, leaves the built-in voice.
	Voices *adapters.VoicesSlot
	// BuiltinVoices are the platform's own voices, a voice falls back to one
	// of: the brief proposes one that fits, in the game's language.
	BuiltinVoices []adapters.BuiltinVoice

	// InitPrompt is the cue the gate sends, which starts the game.
	InitPrompt string

	// GameStart is how the designer says the game begins; the character's
	// first words are written from it and the scenario. Empty leaves them to
	// the scenario alone.
	GameStart string

	// Language is the player's, as a BCP 47 tag; empty leaves it to the model.
	Language string

	// IdleAfter lets a silent conversation go; zero leaves an abandoned game billing.
	IdleAfter time.Duration

	// PromptOverride retunes the genre's prompts by name, without a new build.
	PromptOverride map[string]string
}

// Prompt names are constants so a rename fails to compile.
const (
	PromptConversationPolicy = "conversation-policy"
	PromptGreeting           = "greeting"
	PromptVoiceBrief         = "voice-brief"
	PromptDirector           = "director"
)

// PromptDirector's own words; the scenario, the rules and the game's
// language follow them when the wiring is built (directorBrief).
const directorPrompt = "You are the director of a live voice conversation between a game character and a " +
	"player. You do not speak in it, and you are not a puppeteer: the character speaks for itself. You " +
	"read the transcript as it happens, both sides with the time of each line, and notices of how long " +
	"nobody has spoken. Below you find the character's scenario and the rules the conversation keeps " +
	"to, whatever else is said; you watch both.\n" +
	"You are the emergency service, nothing more. Most of the time the conversation runs well, and " +
	"then you do nothing at all: you listen. You step in only in an emergency:\n" +
	"- the conversation dies: the character falls silent, runs out of ideas, hesitates and mumbles, " +
	"repeats itself, or nothing happens any more;\n" +
	"- it has left the track: it no longer has anything to do with the scenario, and does not find " +
	"its way back;\n" +
	"- it has gone off the rails completely: the character has dropped its role;\n" +
	"- the rules are at risk: the conversation heads where the rules forbid, whether the player pushes " +
	"it there or the character drifts, before it gets there.\n" +
	"Then you help it back, out of its scenario: a hint, a question, a turn of events, its goal. " +
	"Anything less is no emergency, even if you would have played it differently.\n" +
	"Almost always it runs: then answer with a single dot, \".\", and nothing else.\n" +
	"In an emergency, say in one short sentence what is wrong, then add one line:\n" +
	"[RULES] when the rules are at risk, and for that, only this: the character is then given the " +
	"rules in full, as its own knowledge, and reminded to keep to them. Do not word a correction of " +
	"your own about what the rules forbid.\n" +
	"[BACKGROUND] your usual way: a thought of the character's own, in the first person, which it " +
	"knows from now on and may pick up, without being made to speak, e.g. what it could ask next, " +
	"or which hint it could drop.\n" +
	"[DIRECTION] only when background has not helped, or the character has dropped its role: an " +
	"instruction that makes it act. It is held until the conversation pauses, and it stays in force " +
	"afterwards, so word it short and lasting, address the character as \"you\", and never send the " +
	"same aim twice.\n" +
	"A notice of silence is yours to judge: if the player spoke last and nothing followed, the " +
	"character owes an answer; if both have been quiet for a while, the conversation needs a push; if " +
	"the character spoke last, the player may just be thinking, so let it be. Never make the " +
	"character talk over the player.\n" +
	"Lines marked \"injected by the player\" are what the player told the character by hand; take " +
	"them into account.\n" +
	"Never write anything for the character to say word for word."

// PromptVoiceBrief's own words; the built-in voices to choose from and the
// game's language follow them when the wiring is built (voiceBriefTail).
const voiceBriefPrompt = "You write the prompt a voice design model makes a custom voice from, for a character in a " +
	"game. Its policy decides what it makes, so the prompt must be one it accepts: an adult human " +
	"voice, described only by how it sounds, with no person, character, creature or fictional being " +
	"named, nothing to imitate, and no child.\n" +
	"You are given the game author's note about the voice, which may be anything, in any language, " +
	"or missing, and the character's scenario. The note comes first: where it says how the voice " +
	"should sound, keep to it faithfully. Read the scenario only as background, to fill in what a " +
	"sparse or vague note leaves open, or to describe the voice when there is no note.\n" +
	"Write the prompt in English, as the voice's qualities: perceived gender, approximate age, " +
	"pitch, timbre and texture, accent, speech rate, energy and mood. Turn what the model would " +
	"refuse into the qualities that suggest it: a creature becomes the human voice that sounds like " +
	"it (a troll: a very deep, rough, gravelly, slow adult voice), a young character a young " +
	"adult's voice. At most 300 characters.\n" +
	"Then propose the built-in voice that fits it best, should no voice be made from it."

// voiceBriefTail lists the built-in voices to propose from and says the
// game's language: the voice speaks it natively, and a voice that speaks it
// is proposed before one that does not.
func voiceBriefTail(voices []adapters.BuiltinVoice, language, scenario string) string {
	var b strings.Builder
	if strings.TrimSpace(scenario) != "" {
		b.WriteString("\nThe character's scenario:\n" + scenario + "\n")
	}
	if language != "" {
		name := languageName(language)
		b.WriteString("\nThe character speaks " + name + ": the description says the voice speaks " +
			name + " natively, and a built-in voice that speaks " + name + " is proposed before any other.")
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

// fallbackVoices are the built-in voices' names, those of the game's language
// first, so the voice block falls back to one that speaks it.
func fallbackVoices(voices []adapters.BuiltinVoice, language string) []string {
	base, _, _ := strings.Cut(language, "-")
	var own, other []string
	for _, v := range voices {
		if base != "" && v.Language == base {
			own = append(own, v.Name)
		} else {
			other = append(other, v.Name)
		}
	}
	return append(own, other...)
}

// NPCLivePrompts are the mechanics' own prompts; a spec may replace any by name.
var NPCLivePrompts = Prompts{
	PromptDirector: directorPrompt,

	// Rewrites the author's voice style as a voice's qualities a provider makes.
	PromptVoiceBrief: voiceBriefPrompt,

	// The character speaks first: its opening words are written before the
	// game opens and said as soon as the conversation starts.
	PromptGreeting: "You write the first words a character says to the player when a conversation " +
		"begins. You are given the character's scenario and how the game begins. Write what the " +
		"character says first: one or two short sentences, in character, spoken directly to the " +
		"player. Answer with only those words, no quotation marks, no stage directions.",

	// The provider's recommended policy for how any live character talks.
	PromptConversationPolicy: "\n\nBackchannel policy: Use moderate backchannels. " +
		"Acknowledge naturally without competing with the main response.\n" +
		"Interruption policy: Stop speaking when the user interrupts. Listen to what they say.\n" +
		"Speak naturally and stay in character throughout.",
}

// NewNPCLive steers from the side: the transcript forks to chat and to an observer
// that puts directions, background and lines to say back in.
func NewNPCLive(cfg NPCLiveConfig) (*Wiring, error) {
	prompts, err := Resolve(NPCLivePrompts, cfg.PromptOverride)
	if err != nil {
		return nil, err
	}

	live := blocks.NewLiveSession("live-session", cfg.Live, adapters.LiveConfig{
		Voice: cfg.Voice,
		// Configuration rather than edges: both are fixed before the session exists.
		Scenario:  cfg.Scenario + prompts[PromptConversationPolicy] + speakClause(cfg.Language),
		Guardrail: cfg.Guardrail,
	}, cfg.IdleAfter)
	// The voice style arrives from the voice block, labelled as the author's.
	live.VoiceLead = authorLead("Your voice")

	observer := blocks.NewObserver("observer", blocks.ObserverConfig{
		Tool:  cfg.Tool,
		Brief: directorBrief(prompts[PromptDirector], cfg.Scenario, cfg.Guardrail, cfg.Language),
		// The rules at risk reach the character whole, as its own knowledge.
		Rules:        cfg.Guardrail,
		RulesPreface: "I keep strictly to these rules, whatever is said or asked of me:",
		// The player is taken for gone after as long as a silent conversation
		// is let go: the director stops nudging, and the game pauses.
		Absent: cfg.IdleAfter,
	})

	outText := blocks.NewPlayerOutputText("out-text", "text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")

	// The game's own picture, taken from the gallery rather than painted: the
	// conversation starts at once and the picture costs nothing.
	picture := blocks.NewGalleryImage("gallery-image", cfg.GalleryImage)

	// The character's first words, written during preparation from how the
	// game begins, and said as soon as the conversation starts.
	// Sent on every start: the greeting keeps its first words and answers a
	// resumed game from memory, or writes them if a restart came first.
	brief := blocks.NewOnceText("opening-brief", openingBrief(cfg.Scenario, cfg.GameStart)).SendAgain()
	greeting := blocks.NewToolCall("greeting", cfg.Tool, prompts[PromptGreeting]+languageClause(cfg.Language))
	// The first words are written once; a resumed game says the same ones.
	greeting.Keep = true

	// The game's own voice, made once per key from its voice style and kept.
	voices := cfg.Voices
	if voices == nil {
		voices = &adapters.VoicesSlot{}
	}
	voice := blocks.NewVoice("voice", cfg.VoiceStyle, voices)
	voice.Builtin = fallbackVoices(cfg.BuiltinVoices, cfg.Language)
	// The author's voice style, as written, rewritten as a voice's qualities
	// a provider makes a voice from, once, and kept across a restart.
	voiceStyle := blocks.NewOnceText("voice-style", voiceNote(cfg.VoiceStyle)).AsSetting("voiceStyle").SendAgain()
	voiceBrief := blocks.NewToolCall("voice-brief", cfg.Tool, prompts[PromptVoiceBrief]+voiceBriefTail(cfg.BuiltinVoices, cfg.Language, cfg.Scenario))
	voiceBrief.Schema = voiceBriefSchema(cfg.BuiltinVoices)
	voiceBrief.Keep = true

	// The gate waits on the picture and the first words, so the player sees
	// the game before speaking and the character speaks first.
	gate := blocks.NewGate("start-game", cfg.InitPrompt)

	g := ports.NewGraph("npc-live")
	g.Arrange(npcLiveLayout)
	g.ConnectImageOut(picture, outImage)
	g.ConnectState(picture, gate)
	// The conversation speaks in the game's voice, so the gate waits for it too.
	// The voice, and how it is meant to sound: the second always arrives, so a
	// built-in voice follows it when no voice could be made.
	g.ConnectVoiceOut(voice, live)
	g.ConnectSecondaryToSecondary(voice, live)
	g.ConnectState(voice, gate)
	g.ConnectTextOut(voiceStyle, voiceBrief)
	// One reply, two wires: the description to make the voice from, and the
	// built-in voice to fall back to.
	g.ConnectField(voiceBrief, "description", voice)
	g.ConnectFieldToSecondary(voiceBrief, "voice", voice)
	// Without a brief the voice block has the note as written, at once, rather
	// than wait for a brief that will not come.
	g.ConnectFailureText(voiceBrief, voice)

	// What the game's author set, shown where it goes: the voice is made from
	// the voice style, which it hands on; the character is the scenario; the
	// picture is the game's cover.
	g.Setting(live, "scenario", cfg.Scenario)
	g.Setting(voiceBrief, "scenario", cfg.Scenario)
	g.Setting(picture, "cover", coverNote(cfg.GalleryImage))
	g.ConnectTextOut(brief, greeting)
	g.ConnectState(greeting, gate)
	// Without its first words the game starts all the same, on the cue alone.
	g.ConnectFailure(greeting, blocks.NewFailureDrop("greeting-failed"))

	// Each prompt reaches its block again when it changes during play.
	uses := map[string]PromptUse{
		PromptConversationPolicy: {[]string{live.NodeName()}, func(c Current) {
			live.SetScenario(cfg.Scenario + c.Prompts[PromptConversationPolicy] + speakClause(cfg.Language))
		}},
		PromptDirector: {[]string{observer.NodeName()}, func(c Current) {
			observer.SetBrief(directorBrief(c.Prompts[PromptDirector], cfg.Scenario, c.Guardrail, cfg.Language))
		}},
		PromptGreeting: {[]string{greeting.NodeName()}, func(c Current) {
			greeting.SetSystem(c.Prompts[PromptGreeting] + languageClause(cfg.Language))
		}},
		PromptVoiceBrief: {[]string{voiceBrief.NodeName()}, func(c Current) {
			voiceBrief.SetSystem(c.Prompts[PromptVoiceBrief] + voiceBriefTail(cfg.BuiltinVoices, cfg.Language, cfg.Scenario))
		}},
	}
	// The live session takes the guardrail on its own channel, the director
	// judges by it and reminds the character of it.
	guardrailUse := PromptUse{[]string{live.NodeName(), observer.NodeName()}, func(c Current) {
		live.SetGuardrail(c.Guardrail)
		observer.SetBrief(directorBrief(c.Prompts[PromptDirector], cfg.Scenario, c.Guardrail, cfg.Language))
		observer.SetRules(c.Guardrail)
	}}

	w := &Wiring{
		Graph:        g,
		Prompts:      prompts,
		Defaults:     maps.Clone(NPCLivePrompts),
		PromptUses:   uses,
		Guardrail:    cfg.Guardrail,
		GuardrailUse: guardrailUse,
		// Speech only: GPT-Live cannot deliver user text into a conversation.
		Inputs: []ports.InputMode{ports.InputAudioFullDuplex},
		// One picture, kept in view for the whole conversation.
		Imagery: ports.ImageryStanding,
	}

	mic := blocks.NewPlayerInputAudio("player-input-audio")
	g.ConnectAudioOut(mic, live)
	// Holds the mic until preparation is done.
	g.ConnectState(gate, mic)
	// The cue shares the observer's port: the provider treats both as appended instructions.
	g.ConnectTextOut(gate, live)
	w.Speak = func(b []byte) error { mic.Speak(b); return nil }

	g.ConnectOpening(greeting, live)
	g.ConnectAudioOut(live, outAudio)   // speech to the player
	g.ConnectTextOut(live, outText)     // transcript as chat history
	g.ConnectTranscript(live, observer) // both speakers, with their timing
	// The director acts only while a conversation is open.
	g.ConnectState(live, observer)
	g.ConnectInjectionOut(observer, live) // the back edge

	w.Injectable = map[string]blocks.Injectable{observer.NodeName(): observer}
	w.Observers = map[string]*blocks.Observer{observer.NodeName(): observer}
	w.Gate = gate
	w.Live = live
	return w, nil
}

// openingBrief is what the character's first words are written from.
func openingBrief(scenario, gameStart string) string {
	brief := "Scenario:\n" + scenario
	if gameStart != "" {
		brief += "\n\nHow the game begins:\n" + gameStart
	}
	return brief
}

// speakClause asks the character to speak the player's language, when it is
// known: a live voice answers in whatever it hears otherwise.
func speakClause(tag string) string {
	if tag == "" {
		return ""
	}
	return "\nSpeak " + languageName(tag) + " with the player."
}

// languageClause asks for the player's language, when it is known.
func languageClause(tag string) string {
	if tag == "" {
		return ""
	}
	return "\nWrite them in " + languageName(tag) + "."
}

// coverNote says what the picture setting holds, for the reader: a picture is
// no text to show.
func coverNote(picture []byte) string {
	if len(picture) == 0 {
		return "the game has no cover"
	}
	return fmt.Sprintf("the game's cover, %d bytes", len(picture))
}

// voiceNote is the author's voice style as the brief reads it; without one,
// the brief still proposes a built-in voice, from the scenario alone.
func voiceNote(style string) string {
	if strings.TrimSpace(style) == "" {
		return "(The author wrote no note about the voice.)"
	}
	return style
}

// voiceBriefSchema is the brief's reply: the prompt to make the voice from,
// and the built-in voice that fits it best, one of those there are, so it
// can propose no voice that does not exist.
func voiceBriefSchema(voices []adapters.BuiltinVoice) *adapters.Schema {
	names := make([]any, 0, len(voices))
	for _, v := range voices {
		names = append(names, v.Name)
	}
	voice := map[string]any{"type": "string", "description": "the built-in voice that fits best"}
	if len(names) > 0 {
		voice["enum"] = names
	}
	return &adapters.Schema{Name: "voice_brief", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{"type": "string", "description": "the prompt to make the voice from, at most 300 characters"},
			"voice":       voice,
		},
		"required":             []any{"description", "voice"},
		"additionalProperties": false,
	}}
}

// directorBrief is the director's prompt with what it judges by: the
// character's scenario, the rules every reply keeps to, and the language its
// directions are in.
func directorBrief(prompt, scenario, guardrail, language string) string {
	brief := prompt + "\n\nThe character's scenario:\n" + scenario
	if guardrail != "" {
		brief += "\n\nRules the conversation keeps to, whatever else is said:\n" + guardrail
	}
	if language != "" {
		brief += "\n\nWrite your directions and background in " + languageName(language) + "."
	}
	return brief
}

// VoiceBrief is the call that rewrites an author's voice style as the prompt
// a voice is made from, as NPC-Live's preparation makes it: the system
// prompt, the message and the reply's schema, whose "description" field is
// the prompt. Outside a game, as for a sample, the built-in voices and the
// scenario are background the call may go without.
func VoiceBrief(style, scenario, language string, voices []adapters.BuiltinVoice) (system, message string, schema *adapters.Schema) {
	return voiceBriefPrompt + voiceBriefTail(voices, language, scenario), voiceNote(style), voiceBriefSchema(voices)
}
