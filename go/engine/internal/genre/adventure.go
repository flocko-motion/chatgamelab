// Package genre holds one wiring per genre, as Go code so the compiler checks each graph.
package genre

import (
	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// AdventureConfig is what the wiring needs from a session spec, already resolved.
type AdventureConfig struct {
	Tool adapters.Tool

	// Status seeds the values the game keeps, such as Health.
	Status map[string]string

	// Language is the player's, as a BCP 47 tag; empty leaves it to the model.
	Language string

	// PromptOverride retunes the genre's prompts by name, without a new build.
	PromptOverride map[string]string
}

// Prompt names are constants so a rename fails to compile.
const (
	PromptRephrase = "rephrase"
)

// AdventurePrompts are the mechanics' own prompts; a spec may replace any by name.
var AdventurePrompts = Prompts{
	// Third person with the outcome open, so the story decides what happens.
	PromptRephrase: "Rephrase the player's input in third person, making the outcome uncertain. " +
		"Return ONLY the rephrased text, nothing else.\n" +
		"Example: 'I attack the wolf and wrestle him to the ground' → " +
		"'The player attacks the wolf, hoping to wrestle him to the ground.'",
}

// languageClause asks for the player's language; empty asks for nothing.
func languageClause(tag string) string {
	if tag == "" {
		return ""
	}
	return "\n\nKeep the response in the language with BCP 47 tag `" + tag + "`."
}

// NewAdventure is the turn-based genre: rephrase, extract plot, image prompt and
// status in one call, then expand and fan out to media.
func NewAdventure(cfg AdventureConfig) (*Wiring, error) {
	prompts, err := Resolve(AdventurePrompts, cfg.PromptOverride)
	if err != nil {
		return nil, err
	}
	prompts[PromptRephrase] += languageClause(cfg.Language)

	player := blocks.NewPlayerInputText("player-input-text")
	rephrase := blocks.NewToolCall("rephrase", cfg.Tool, prompts[PromptRephrase])
	outline := blocks.NewDummyExtraction("outline")
	expand := blocks.NewDummyThreaded("expand", "prose")
	image := blocks.NewDummyImage("image")
	tts := blocks.NewDummyTTS("tts")

	outText := blocks.NewPlayerOutputText("out-text", "text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")
	outProps := blocks.NewPlayerOutputProps("out-props", "props")
	props := blocks.NewPropsStore("props-store", cfg.Status)

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
		Graph:   g,
		Gate:    gate,
		Prompts: prompts,
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
	}, nil
}
