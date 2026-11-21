// Package genre holds one wiring per genre, as Go code so the compiler checks each graph.
package genre

import (
	"maps"
	"slices"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// AdventureConfig is what the wiring needs from a session spec, already resolved.
type AdventureConfig struct {
	Tool adapters.Tool
	// Thread runs the story model's conversation.
	Thread adapters.Thread

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
	PromptOutline  = "outline"
	PromptExpand   = "expand"
)

// AdventurePrompts are the mechanics' own prompts; a spec may replace any by name.
var AdventurePrompts = Prompts{
	// Third person with the outcome open, so the story decides what happens.
	PromptRephrase: "Rephrase the player's input in third person, making the outcome uncertain. " +
		"Return ONLY the rephrased text, nothing else.\n" +
		"Example: 'I attack the wolf and wrestle him to the ground' → " +
		"'The player attacks the wolf, hoping to wrestle him to the ground.'",
	// A placeholder, overridable by name.
	PromptOutline: "You run a text adventure. You are given the player's action, told in third " +
		"person with its outcome open, and the game's current status. Decide what plausibly " +
		"happens as the world reacts, true to everything told so far. Answer with the outline " +
		"of what happens in a few plain sentences, a prompt for a picture of the scene, and " +
		"every status field with its value after this turn, changed or not.",
	// A first draft, overridable by name.
	PromptExpand: "You narrate a text adventure. Turn the outline you are given into the next " +
		"passage of the story: vivid second-person prose, a few sentences long, true to " +
		"everything told so far. Narrate only what the outline says happens; never decide " +
		"what the player does next.",
}

// languageClause asks for the player's language; empty asks for nothing.
func languageClause(tag string) string {
	if tag == "" {
		return ""
	}
	return "\n\nKeep the response in the language with BCP 47 tag `" + tag + "`."
}

// The fields of Outline's reply.
const (
	outlineText        = "outline"
	outlineImagePrompt = "image_prompt"
	outlineStatus      = "status"
)

// outlineResult feeds Outline's outputs from its reply.
var outlineResult = blocks.Result{Text: outlineText, SecondaryText: outlineImagePrompt, Props: outlineStatus}

// OutlineSchema is the shape of Outline's reply, a call parameter of the
// genre's: the outline, the image prompt, and the status, one property per
// status field of the session, every one in every reply. A game without status
// fields has no status in its schema. Strict mode allows no free-form map, so
// the fields are listed.
func OutlineSchema(status map[string]string) adapters.Schema {
	text := map[string]any{"type": "string"}
	properties := map[string]any{outlineText: text, outlineImagePrompt: text}
	required := []string{outlineText, outlineImagePrompt}
	if len(status) > 0 {
		fields := map[string]any{}
		names := slices.Sorted(maps.Keys(status))
		for _, name := range names {
			fields[name] = text
		}
		properties[outlineStatus] = map[string]any{
			"type":                 "object",
			"properties":           fields,
			"required":             names,
			"additionalProperties": false,
		}
		required = append(required, outlineStatus)
	}
	return adapters.Schema{Name: "outline", Body: map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}}
}

// NewAdventure is the turn-based genre: rephrase, then Outline decides the
// plot, image prompt and status in one structured call, then Expand writes
// the prose on the same thread and the turn fans out to media.
func NewAdventure(cfg AdventureConfig) (*Wiring, error) {
	prompts, err := Resolve(AdventurePrompts, cfg.PromptOverride)
	if err != nil {
		return nil, err
	}
	prompts[PromptRephrase] += languageClause(cfg.Language)
	// The status values reach the player, so they are in the player's language.
	prompts[PromptOutline] += languageClause(cfg.Language)
	prompts[PromptExpand] += languageClause(cfg.Language)

	player := blocks.NewPlayerInputText("player-input-text")
	rephrase := blocks.NewToolCall("rephrase", cfg.Tool, prompts[PromptRephrase])
	// Outline takes the thread's first seat, Expand the second.
	story := blocks.NewThread("story-thread", cfg.Thread)
	outline := blocks.NewStructured("outline", story, prompts[PromptOutline], OutlineSchema(cfg.Status), outlineResult)
	expand := blocks.NewThreaded("expand", story, prompts[PromptExpand])
	image := blocks.NewDummyImage("image")
	tts := blocks.NewDummyTTS("tts")

	outText := blocks.NewPlayerOutputStream("out-text", "text")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")
	outProps := blocks.NewPlayerOutputProps("out-props", "props")
	props := blocks.NewPropsStore("props-store", cfg.Status)

	// No opening line: Adventure begins when the player acts.
	gate := blocks.NewGate("start-game", "")

	g := ports.NewGraph("adventure")
	g.Track(gate)
	// The thread is exported once, under its own name, for both its blocks.
	g.Share(story)
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

	// The player watches the prose appear; narration waits for the whole.
	g.ConnectStreamOut(expand, outText)
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
