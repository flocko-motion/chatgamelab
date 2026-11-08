package genre

import (
	"strings"
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

	// Scenario and Guardrail stay apart: nothing a designer writes may reach the guardrail.
	Scenario  string
	Guardrail string
	Voice     string

	// ImageStyle is the designer's look; the genre decides the framing.
	ImageStyle string

	// InitPrompt is the cue the gate sends, which starts the game.
	InitPrompt string

	// IdleAfter lets a silent conversation go; zero leaves an abandoned game billing.
	IdleAfter time.Duration

	// MuteObserver keeps the observer on the graph but judging nothing.
	MuteObserver bool

	// PromptOverride retunes the genre's prompts by name, without a new build.
	PromptOverride map[string]string
}

// Prompt names are constants so a rename fails to compile.
const (
	PromptPortrait           = "portrait"
	PromptConversationPolicy = "conversation-policy"
	PromptObserver           = "observer"
)

// NPCLivePrompts are the mechanics' own prompts; a spec may replace any by name.
var NPCLivePrompts = Prompts{
	// Lives here because the image block also draws Adventure's scenes.
	PromptPortrait: "Paint a portrait of the character described below. " +
		"Head and shoulders, facing the viewer, no text or lettering.\n\n",

	// The provider's recommended policy for how any live character talks.
	PromptConversationPolicy: "\n\nBackchannel policy: Use moderate backchannels. " +
		"Acknowledge naturally without competing with the main response.\n" +
		"Interruption policy: Stop speaking when the user interrupts. Listen to what they say.\n" +
		"Speak naturally and stay in character throughout.",

	// What the observer judges each line against.
	PromptObserver: "You judge one line spoken by a character in a game.\n" +
		"Answer VIOLATION followed by a short reason, or OK.\n" +
		"Judge three things: does the line breach the constraint below, is the character " +
		"drifting out of the role the scenario gives them, and is it turning sycophantic.",
}

// DefaultImageStyle is stated because, left to choose, the model varies every time.
const DefaultImageStyle = "painterly, warm light, soft brushwork"

// styleClause goes last, where an image model reads it as treatment of the whole.
func styleClause(style string) string {
	if strings.TrimSpace(style) == "" {
		style = DefaultImageStyle
	}
	return "\n\nStyle: " + style
}

// NewNPCLive steers from the side: the transcript forks to chat and to an observer
// whose corrections feed back. Its findings stay unwired, internal to the story.
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

	observer := blocks.NewObserver("observer", cfg.Tool, prompts[PromptObserver], cfg.Guardrail, cfg.Scenario, 2)
	if cfg.MuteObserver {
		observer.Mute()
	}

	outText := blocks.NewPlayerOutputText("out-text", "text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")

	// The init stem's head: constant, but drawn as a pipeline because it becomes a picture.
	portraitPrompt := blocks.NewOnceText("portrait-prompt",
		prompts[PromptPortrait]+cfg.Scenario+styleClause(cfg.ImageStyle))
	portrait := blocks.NewImage("portrait", cfg.Image)

	// The gate waits on the portrait so the player sees who they address before speaking.
	gate := blocks.NewGate("start-game", cfg.InitPrompt)

	g := ports.NewGraph("npc-live")
	g.ConnectTextOut(portraitPrompt, portrait)
	g.ConnectImageOut(portrait, outImage)
	g.ConnectState(portrait, gate)

	// Speech only: GPT-Live cannot deliver user text into a conversation.
	w := &Wiring{
		Graph:   g,
		Prompts: prompts,
		Inputs:  []ports.InputMode{ports.InputAudioFullDuplex},
		// One portrait, kept in view for the whole conversation.
		Imagery: ports.ImageryStanding,
	}

	mic := blocks.NewPlayerInputAudio("player-input-audio")
	g.ConnectAudioOut(mic, live)
	// Holds the mic until preparation is done.
	g.ConnectState(gate, mic)
	// The cue shares the observer's port: the provider treats both as appended instructions.
	g.ConnectTextOut(gate, live)
	w.Speak = func(b []byte) { mic.Speak(b) }

	g.ConnectAudioOut(live, outAudio) // speech to the player
	g.ConnectTextOut(live, outText)   // transcript as chat history
	g.ConnectTextOut(live, observer)  // same source, independent stream
	g.ConnectTextOut(observer, live)  // the back edge

	w.Observer = observer
	w.Gate = gate
	w.Live = live
	// Only new pictures stop: the portrait, once made, stays.
	w.Switches = map[adapters.Toggle][]ports.Switchable{adapters.ToggleImages: {portrait}}
	return w, nil
}
