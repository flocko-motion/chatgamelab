// Package genre holds one wiring per genre. A genre is Go code, not data: the
// graph below is checked by the compiler on every build, and by Validate for
// the completeness the type system cannot see.
package genre

import (
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NewAdventure is the turn-based genre: rephrase, then one extraction call
// yielding plot, image prompt and status at once, then expand and the media
// fan-out.
//
// The four output blocks it wires are its event schema: text, audio, image,
// props.
func NewAdventure(status map[string]string) *Wiring {
	player := blocks.NewPlayerInputText("player-input-text")
	rephrase := blocks.NewDummyToolCall("rephrase", "3rd-person")
	outline := blocks.NewDummyExtraction("outline")
	expand := blocks.NewDummyThreaded("expand", "prose")
	image := blocks.NewDummyImage("image")
	tts := blocks.NewDummyTTS("tts")

	outText := blocks.NewPlayerOutputText("out-text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio")
	outImage := blocks.NewPlayerOutputImage("out-image")
	outProps := blocks.NewPlayerOutputProps("out-props")
	props := blocks.NewPropsStore("props-store", status)

	// No opening line yet: Adventure begins when the player acts. v1 generates
	// an opening scene from an init prompt, and this is where that goes.
	gate := blocks.NewGate("start-game", "")

	g := ports.NewGraph("adventure")
	g.Track(gate)
	// Adventure prepares nothing yet, so its gate opens at once — but the edge
	// is wired, so the lifecycle is the same shape in every genre and adding a
	// preparation step later changes one line rather than the design.
	g.ConnectState(gate, player)
	g.ConnectTextOut(player, rephrase)
	g.ConnectTextOut(rephrase, outline)

	g.ConnectTextOut(outline, expand)
	// The second text output of the same block. Distinct method, distinct edge:
	// these two lines cannot be confused for one another at a call site.
	g.ConnectSecondaryTextOut(outline, image)
	// Status values travel on an edge rather than living in the model's memory:
	// the store holds them, hands them back before each turn, and is the one
	// place they can be seeded or corrected.
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
		Say:   player.Say,
		Sinks: map[string]chan string{
			"text":  outText.Seen,
			"audio": outAudio.Seen,
			"image": outImage.Seen,
			"props": outProps.Seen,
		},
	}
}
