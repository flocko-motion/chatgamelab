package genre

import (
	"time"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NPCLiveConfig is what the wiring needs from a session spec, already resolved.
type NPCLiveConfig struct {
	Live  adapters.Live
	Tool  adapters.Tool
	Image adapters.Image

	// Scenario and Guardrail have different authors and stay apart because of
	// it: a game designer writes the first, the platform's cascade resolves the
	// second, and nothing a designer can write may reach the second.
	Scenario  string
	Guardrail string
	Voice     string

	// InitPrompt is the cue the gate sends, which starts the game.
	InitPrompt string

	// IdleAfter is how long the conversation may go unspoken before it is let
	// go. Zero leaves it open, which on a genre billed by the minute means an
	// abandoned game costs money until somebody notices.
	IdleAfter time.Duration

	// MuteObserver runs the observer as a node that judges nothing, for proving
	// the voice path before a second model joins it.
	MuteObserver bool

	// PromptOverride replaces one of the genre's own prompts by name, so a
	// session can be retuned without a new build.
	PromptOverride map[string]string
}

// The names of NPC-Live's own prompts. Constants rather than bare strings at
// the call site, so a rename is a compile error instead of a session quietly
// running on a default.
const (
	PromptPortrait           = "portrait"
	PromptConversationPolicy = "conversation-policy"
	PromptObserver           = "observer"
)

// NPCLivePrompts is what the mechanics need to work at all, as opposed to what
// any particular game is about. A spec may replace any of them by name.
var NPCLivePrompts = Prompts{
	// Turns a scenario into a request for a picture. The image block knows
	// nothing about portraits, because Adventure's illustrations are scenes.
	PromptPortrait: "Paint a portrait of the character described below. " +
		"Head and shoulders, facing the viewer, painterly, no text or lettering.\n\n",

	// What the provider's prompting guide asks every live character to be
	// given. It shapes how a character talks rather than who they are, which is
	// why it is ours and the scenario is not.
	PromptConversationPolicy: "\n\nBackchannel policy: Use moderate backchannels. " +
		"Acknowledge naturally without competing with the main response.\n" +
		"Interruption policy: Stop speaking when the user interrupts. Listen to what they say.\n" +
		"Speak naturally and stay in character throughout.",

	// What the observer looks for. Drift shows across several exchanges rather
	// than in one line, which is why the block keeps a thread rather than
	// scoring each line cold.
	PromptObserver: "You judge one line spoken by a character in a game.\n" +
		"Answer VIOLATION followed by a short reason, or OK.\n" +
		"Judge three things: does the line breach the constraint below, is the character " +
		"drifting out of the role the scenario gives them, and is it turning sycophantic.",
}

// NewNPCLive puts nothing in the conversation's path. Control comes from the
// side: the live block's transcript forks to the player's chat history and to
// an observer, whose correction feeds back into the live block. That back edge
// is a cycle, and it is the point.
//
// It wires three output blocks — audio, text and image — and those are its
// event schema. The observer's findings are wired nowhere: it is an internal
// control on the story, and what a player sees is the character being corrected
// rather than a note saying so. An output with no edge reaches nothing.
//
// There is no typed input, because GPT-Live has no event that delivers user
// text into a conversation.
func NewNPCLive(cfg NPCLiveConfig) (*Wiring, error) {
	prompts, err := Resolve(NPCLivePrompts, cfg.PromptOverride)
	if err != nil {
		return nil, err
	}

	live := blocks.NewLiveSession("live-session", cfg.Live, adapters.LiveConfig{
		Voice: cfg.Voice,
		// The character, and the constraint on the character. Configuration
		// rather than edges: both are fixed before the session exists and
		// neither ever changes, and a port would say otherwise.
		Scenario:  cfg.Scenario + prompts[PromptConversationPolicy],
		Guardrail: cfg.Guardrail,
	}, cfg.IdleAfter)

	observer := blocks.NewObserver("observer", cfg.Tool, prompts[PromptObserver], cfg.Guardrail, cfg.Scenario, 2)
	if cfg.MuteObserver {
		observer.Mute()
	}

	outText := blocks.NewPlayerOutputText("out-text", "text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")

	// The head of the init stem. Its value is constant, but what follows it is
	// a flow — a prompt becomes a picture — and that processing is why the
	// stem is drawn as a pipeline rather than folded into configuration.
	portraitPrompt := blocks.NewOnceText("portrait-prompt", prompts[PromptPortrait]+cfg.Scenario)
	portrait := blocks.NewImage("portrait", cfg.Image)

	// The gate both releases the player and sends the first cue. It gates on
	// the portrait: a player should see who they are talking to before they
	// speak, and a face arriving mid-sentence is worse than a short wait. The
	// cost is a pause before the first word rather than a delayed session,
	// since the conversation is invited only once the gate opens anyway.
	gate := blocks.NewGate("start-game", cfg.InitPrompt)

	g := ports.NewGraph("npc-live")
	g.ConnectTextOut(portraitPrompt, portrait)
	g.ConnectImageOut(portrait, outImage)
	g.ConnectState(portrait, gate)

	// Spoken, and only spoken: GPT-Live has no event that delivers user text
	// into a conversation, so a text box here would be a control that cannot
	// work.
	w := &Wiring{
		Graph:   g,
		Prompts: prompts,
		Inputs:  []ports.InputMode{ports.InputAudioFullDuplex},
		// One portrait, made once, of the character being spoken to. It stays
		// in view for the conversation's lifetime rather than scrolling off
		// with the dialogue.
		Imagery: ports.ImageryStanding,
	}

	mic := blocks.NewPlayerInputAudio("player-input-audio")
	g.ConnectAudioOut(mic, live)
	// The gate holds the player until preparation is done. Without this edge
	// the gate would be decoration: something that opens while the game has
	// already started is not a gate.
	g.ConnectState(gate, mic)
	// The cue, appended to the character's standing instructions once the game
	// begins. It shares a port with the observer's corrections because the
	// provider treats both the same way.
	g.ConnectTextOut(gate, live)
	w.Speak = func(b []byte) { mic.Speak(b) }

	g.ConnectAudioOut(live, outAudio) // speech to the player
	g.ConnectTextOut(live, outText)   // transcript as chat history
	g.ConnectTextOut(live, observer)  // same source, independent stream
	g.ConnectTextOut(observer, live)  // the back edge

	w.Observer = observer
	w.Gate = gate
	w.Live = live
	return w, nil
}
