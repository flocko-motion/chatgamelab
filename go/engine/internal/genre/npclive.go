package genre

import (
	"fmt"
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
)

// NPCLivePrompts are the mechanics' own prompts; a spec may replace any by name.
var NPCLivePrompts = Prompts{
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
		Scenario:  cfg.Scenario + prompts[PromptConversationPolicy],
		Guardrail: cfg.Guardrail,
	}, cfg.IdleAfter)
	// The voice style arrives from the voice block, labelled as the author's.
	live.VoiceLead = authorLead("Your voice")

	observer := blocks.NewObserver("observer")

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

	// What the game's author set, shown where it goes: the voice is made from
	// the voice style, which it hands on; the character is the scenario; the
	// picture is the game's cover.
	g.Setting(voice, "voiceStyle", cfg.VoiceStyle)
	g.Setting(live, "scenario", cfg.Scenario)
	g.Setting(picture, "cover", coverNote(cfg.GalleryImage))
	g.ConnectTextOut(brief, greeting)
	g.ConnectState(greeting, gate)
	// Without its first words the game starts all the same, on the cue alone.
	g.ConnectFailure(greeting, blocks.NewFailureDrop("greeting-failed"))

	// Speech only: GPT-Live cannot deliver user text into a conversation.
	w := &Wiring{
		Graph:   g,
		Prompts: prompts,
		Inputs:  []ports.InputMode{ports.InputAudioFullDuplex},
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
	g.ConnectAudioOut(live, outAudio)     // speech to the player
	g.ConnectTextOut(live, outText)       // transcript as chat history
	g.ConnectTextOut(live, observer)      // same source, independent stream
	g.ConnectInjectionOut(observer, live) // the back edge

	w.Injectable = map[string]blocks.Injectable{observer.NodeName(): observer}
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
