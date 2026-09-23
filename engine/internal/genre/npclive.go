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

	// One portrait of the character, made once and never again. It takes no
	// input edge: its prompt is configuration, and nothing flowing through the
	// graph triggers it.
	//
	// Its done signal is deliberately not wired to the gate. A conversation can
	// start before the picture exists, and holding a player in silence while an
	// image renders would be the wrong trade.
	portrait := blocks.NewImageOnce("portrait", cfg.Image, cfg.Scenario)
	gate := blocks.NewGate("start-game")

	g := ports.NewGraph("npc-live")
	g.ConnectImageOut(portrait, outImage)

	// The observer's flags ride the same event stream. That also makes them the
	// one thing worth persisting from an otherwise ephemeral conversation:
	// what was flagged, without keeping the dialogue.
	// The gate has no gating inputs for this genre, so it opens at once. It is
	// still wired so the lifecycle is the same shape in every genre.
	g.Track(gate)

	w := &Wiring{Graph: g, Sinks: map[string]chan string{
		"text":  outText.Seen,
		"audio": outAudio.Seen,
		"image": outImage.Seen,
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
	w.Gate = gate
	return w
}
