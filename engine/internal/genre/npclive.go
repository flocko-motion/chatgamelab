package genre

import (
	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NPCLiveConfig is what the wiring needs from a session spec, already resolved.
type NPCLiveConfig struct {
	Live      adapters.Live
	Tool      adapters.Tool
	Image     adapters.Image
	LiveCfg   adapters.LiveConfig
	Guardrail string
	Scenario  string
	// InitPrompt is the first message the gate sends, which starts the game.
	InitPrompt string

	// Script drives a dummy input instead of a live player, for tests and the
	// standalone server.
	Script blocks.InputScript
}

// NewNPCLive puts nothing in the conversation's path. Control comes from the
// side: the live block's transcript forks to the player's chat history and to
// an observer, whose correction feeds back into the live block. That back edge
// is a cycle, and it is the point.
//
// It wires two output blocks — audio and text — so its event schema is those
// two and nothing else. The player's own speech is never transcribed for the
// record.
func NewNPCLive(cfg NPCLiveConfig) *Wiring {
	live := blocks.NewLiveSession("live-session", cfg.Live, cfg.LiveCfg)
	observer := blocks.NewObserver("observer", cfg.Tool, cfg.Guardrail, cfg.Scenario, 2)

	outText := blocks.NewPlayerOutputText("out-text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio")
	outImage := blocks.NewPlayerOutputImage("out-image")

	// One portrait of the character, made once and never again — and once
	// because the source fires once, not because the block refuses a second.
	//
	// It gates the game: the player should see who they are talking to before
	// they speak, and a face arriving mid-sentence is worse than a short wait.
	// Gating costs the player a pause before their first word rather than
	// delaying the session, since the live connection opens regardless.
	scenarioPrompt := blocks.NewOnceText("scenario-prompt", cfg.Scenario)

	portrait := blocks.NewImage("portrait", cfg.Image)
	// The gate both releases the player and sends the first message: the
	// scenario, then the cue to act on it. The scenario also stands as the
	// session's instruction, so the character carries it whether it reads the
	// opening message or the instruction it was given.
	gate := blocks.NewGate("start-game", opening(cfg.Scenario, cfg.InitPrompt))

	g := ports.NewGraph("npc-live")
	g.ConnectTextOut(scenarioPrompt, portrait)
	g.ConnectImageOut(portrait, outImage)
	g.ConnectState(portrait, gate)

	// The observer's flags ride the same event stream. That also makes them the
	// one thing worth persisting from an otherwise ephemeral conversation:
	// what was flagged, without keeping the dialogue.
	w := &Wiring{Graph: g, Sinks: map[string]chan string{
		"text":  outText.Seen,
		"audio": outAudio.Seen,
		"image": outImage.Seen,
		"flag":  observer.Flags,
	}}

	// Both player inputs are always wired, because a live session accepts both.
	// Typing is how the genre is played without a microphone, and how someone
	// who would rather not speak can still play.
	mic := blocks.NewPlayerInputAudio("player-input-audio")
	keyboard := blocks.NewPlayerInputText("player-input-text")
	g.ConnectAudioOut(mic, live)
	g.ConnectTextOut(keyboard, live)
	// The gate holds the player until preparation is done. Without these edges
	// the gate would be decoration: something that opens while the game has
	// already started is not a gate.
	g.ConnectState(gate, mic)
	g.ConnectState(gate, keyboard)
	g.ConnectTextOut(gate, live)
	w.Speak = func(b []byte) { mic.Speak(b) }
	w.Say = keyboard.Say

	// A script drives the conversation in addition to the player rather than
	// instead of them: a self-playing session that ignores whoever is watching
	// is only useful to a test, and even a test may want to interject.
	if len(cfg.Script.Lines) > 0 {
		scripted := blocks.NewDummyInputAudio("scripted-input-audio", cfg.Script)
		g.ConnectAudioOut(scripted, live)
	}

	g.ConnectAudioOut(live, outAudio)           // speech to the player
	g.ConnectTextOut(live, outText)             // transcript as chat history
	g.ConnectTextOut(live, observer)            // same source, independent stream
	g.ConnectTextOutToSecondary(observer, live) // the back edge

	w.Observer = observer
	w.Gate = gate
	return w
}

// opening is what the gate says to begin the game: who the character is,
// followed by what to do about it.
func opening(scenario, initPrompt string) string {
	switch {
	case scenario == "":
		return initPrompt
	case initPrompt == "":
		return scenario
	default:
		return scenario + "\n\n" + initPrompt
	}
}
