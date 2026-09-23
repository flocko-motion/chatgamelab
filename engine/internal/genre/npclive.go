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
	LiveCfg   adapters.LiveConfig
	Guardrail string
	Scenario  string

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

	g := ports.NewGraph("npc-live")

	// The observer's flags ride the same event stream. That also makes them the
	// one thing worth persisting from an otherwise ephemeral conversation:
	// what was flagged, without keeping the dialogue.
	w := &Wiring{Graph: g, Sinks: map[string]chan string{
		"text":  outText.Seen,
		"audio": outAudio.Seen,
		"flag":  observer.Flags,
	}}

	if len(cfg.Script.Lines) > 0 {
		scripted := blocks.NewDummyInputAudio("dummy-input-audio", cfg.Script)
		g.ConnectAudioOut(scripted, live)
	} else {
		player := blocks.NewPlayerInputAudio("player-input-audio")
		g.ConnectAudioOut(player, live)
		w.Speak = func(b []byte) { player.Speak(b) }
	}

	g.ConnectAudioOut(live, outAudio) // speech to the player
	g.ConnectTextOut(live, outText)   // transcript as chat history
	g.ConnectTextOut(live, observer)  // same source, independent stream
	g.ConnectTextOut(observer, live)  // the back edge

	w.Observer = observer
	return w
}
