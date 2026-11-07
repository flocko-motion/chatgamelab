// Package genre holds one wiring per genre, as Go code so the compiler checks each graph.
package genre

import (
	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NewAdventure is the turn-based genre: rephrase, extract plot, image prompt and
// status in one call, then expand and fan out to media.
func NewAdventure(status map[string]string) *Wiring {
	player := blocks.NewPlayerInputText("player-input-text")
	rephrase := blocks.NewDummyToolCall("rephrase", "3rd-person")
	outline := blocks.NewDummyExtraction("outline")
	expand := blocks.NewDummyThreaded("expand", "prose")
	image := blocks.NewDummyImage("image")
	tts := blocks.NewDummyTTS("tts")

	outText := blocks.NewPlayerOutputText("out-text", "text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")
	outProps := blocks.NewPlayerOutputProps("out-props", "props")
	props := blocks.NewPropsStore("props-store", status)

	// No opening line: Adventure begins when the player acts.
	gate := blocks.NewGate("start-game", "")

	g := ports.NewGraph("adventure")
	g.Track(gate)
	// Wired though it opens at once, so every genre has the same lifecycle shape.
	g.ConnectState(gate, player)
	g.ConnectTextOut(player, rephrase)
	g.ConnectTextOut(rephrase, outline)

	g.ConnectTextOut(outline, expand)
	// The block's second text output, on its own method so the two edges stay distinct.
	g.ConnectSecondaryTextOut(outline, image)
	// The store holds status values and is the one place to seed or correct them.
	g.ConnectPropsOut(outline, props)
	g.ConnectPropsOut(props, outline)
	g.ConnectPropsOut(props, outProps)

	g.ConnectTextOut(expand, outText)
	g.ConnectTextOut(expand, tts)
	g.ConnectAudioOut(tts, outAudio)
	g.ConnectImageOut(image, outImage)

	return &Wiring{
		Graph: g,
		Gate:  gate,
		// The graph's shape is fixed; a toggle that is off stops its block working.
		Switches: map[adapters.Toggle][]ports.Switchable{
			adapters.ToggleImages:   {image},
			adapters.ToggleAudioOut: {tts},
		},
		Say: player.Say,
		// Typed only until Adventure wires push-to-talk transcription.
		Inputs: []ports.InputMode{ports.InputText},
		// A picture per turn, each belonging to the scene it illustrates.
		Imagery: ports.ImageryPerTurn,
	}
}
