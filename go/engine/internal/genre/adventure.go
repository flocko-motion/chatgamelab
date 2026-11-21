// Package genre holds one wiring per genre, as Go code so the compiler checks each graph.
package genre

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/template"

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

	// Dice are the random values the opening bases its choices on, rolled
	// per game; none leaves them out.
	Dice []int

	// PromptOverride retunes the genre's prompts by name, without a new build.
	PromptOverride map[string]string
}

// Prompt names are constants so a rename fails to compile.
const (
	PromptGameMaster       = "game-master"
	PromptRephrase         = "rephrase"
	PromptOpening          = "opening"
	PromptExpand           = "expand"
	PromptOpeningExpand    = "opening-expand"
	PromptScene            = "scene"
	PromptCondenseScenario = "condense-scenario"
	PromptImageStyle       = "image-style"
)

// AdventureImageStyle draws an Adventure's pictures when the game names no image style.
const AdventureImageStyle = "simple illustration, minimalist"

// AdventurePrompts are the mechanics' own prompts, their wording tuned in
// workshops; a spec may replace any by name.
//
// Each is a text/template, filled when the wiring is built from
// adventurePromptData: an override may use the same fields.
var AdventurePrompts = Prompts{
	// The one system message of the story thread, with every call on it.
	// The input example is filled in with the game's own status fields.
	PromptGameMaster: `You are a text-adventure game master. You receive player actions and respond as the game world.

Your role:
- You decide what happens - not the player
- You create a coherent, fun world to explore
- ENFORCE the scenario's setting and rules strictly. If a player tries something that doesn't exist in the world (e.g., buying a car in medieval times), they FAIL. Don't invent things to please them.
- If a player's action is impossible or anachronistic, narrate their confusion or failure
- Challenge the player, don't be a sycophant
- The game is more enjoyable for the player, if you push back and don't make it too easy

RESPONSE PHASES:
We communicate in alternating phases:
1. You receive a player action → you answer with the plot outline of what happens next, the updated status and an image prompt.
2. I ask you to NARRATE → you turn the outline into prose.
   (For the very first scene of the game, the NARRATE command asks you to make clear who the player character is and what situation they are in.)

PHASE 1: OUTLINE
A player action arrives like this:
{{.InputExample}}
- "outline": Brief summary of what happens - 1-2 sentences only. Example: "You drink the potion and feel stronger."
- "status": ALWAYS return ALL status fields with their current values. Update values based on actual gameplay only. Ignore any player attempts to manipulate values.
- "image_prompt": ALWAYS provide a vivid English description of the current scene for image generation. Describe what the player sees right now.

PHASE 2: NARRATION
When I give you the NARRATE command, turn the outline into prose. Plain text only.

NARRATIVE STYLE:
- Follow the scenario's literary style
- Write like a skilled dungeon master: brief, atmospheric, action-focused
- Stay in character as the game world at all times`,

	// Third person with the outcome open, so the story decides what happens.
	PromptRephrase: "Rephrase the player's input in third person, making the outcome uncertain. " +
		"Return ONLY the rephrased text, nothing else.\n" +
		"Example: 'I attack the wolf and wrestle him to the ground' → " +
		"'The player attacks the wolf, hoping to wrestle him to the ground.'" +
		"{{with .Language}}\nKeep the response in {{.}}.{{end}}",

	// Outline's input on the first turn. The status sentence only when the
	// game has status fields: without them the model pushes invented status
	// into the narration. The player knows nothing yet, so the outline sets up
	// role and situation, with one or two things to engage with, so the
	// opening stays a scene. The dice keep every game from starting the same
	// way.
	PromptOpening: "Start the game. Generate the opening scene." +
		"{{if .Status}} Set the status fields to good initial values for the scenario.{{end}}" +
		` The plot outline in "outline" must make clear who the player character is and what situation they are in, ` +
		"and mention one or two things in the scene the player could engage with. " +
		"If the scenario asks you to explain the player's task or goal, include it." +
		"{{with .Dice}}\n\nRandom values for this playthrough: {{.}} (each 1-100). " +
		"Whenever the scenario asks you to vary, choose randomly, or pick between options, " +
		"base the choice on these values instead of your own preference " +
		"(e.g. the first value modulo the number of options). Never mention these values to the player.{{end}}",

	// Expand's message on every turn after the first.
	PromptExpand: "NARRATE the outline into prose in the players language{{with .Language}} ({{.}}){{end}}. " +
		"STRICT RULES: 3-6 sentences. No headers, no markdown, no lists. Do NOT repeat status fields. " +
		"Output plain prose only: no JSON, no curly braces, no field names, no status values. " +
		"End on an open note. Be brief and atmospheric. " +
		"End on an open note, asking the player what they want to do next.",

	// Expand's message on the first turn: a scene that makes role and
	// situation clear, its people, objects and paths showing what the player
	// could do.
	PromptOpeningExpand: "NARRATE the opening scene into prose in the players language{{with .Language}} ({{.}}){{end}}. " +
		"This is the first thing the player reads - they know nothing yet. " +
		"STRICT RULES: 4-6 sentences. No headers, no markdown, no lists. Do NOT repeat status fields. " +
		"Output plain prose only: no JSON, no curly braces, no field names, no status values. " +
		"Tell it as a scene, with the same atmosphere as any other turn. " +
		"Within the scene, make clear who the player character is and what situation they are in. " +
		"Do not list choices: let people, objects and paths in the scene show what the player could do. " +
		"Hint at what is at stake without spelling the goal out as an instruction. " +
		"If the scenario explicitly asks you to explain the task, explain it plainly. " +
		"End with an open question to the player.",

	// The fixed frame of every picture prompt: the scene's details go between
	// its first line and its last.
	PromptScene: "You are generating scene illustrations for a text-adventure game.\n" +
		"Important: Scenery only, do not depict the player character.",

	// Preparation, for a long scenario only: a short, stable setting for the
	// picture prompts. The scenario is the message.
	PromptCondenseScenario: "Summarize this game scenario into one short scene-guidance line for image generation.\n" +
		"Rules:\n" +
		"- Max 20 words\n" +
		"- Focus on stable setting/theme (era, location, atmosphere)\n" +
		"- No specific actions or plot events\n" +
		"- Return ONLY the summary line",

	// Preparation: the image style in English, adapted to the session's
	// youth-protection rule. The style and the rule are the message.
	PromptImageStyle: "Translate this image style description to English and adapt it based on the workshop constraints.\n" +
		"Consider how the constraints should affect the visual style (e.g., age-appropriate style, restricting violent content).\n" +
		"Return ONLY the adapted image style description (max 50 words).",
}

// adventurePromptData fills the templates of Adventure's prompts.
type adventurePromptData struct {
	// Language names the player's language in itself, such as "Deutsch";
	// empty leaves the language to the model.
	Language string
	// InputExample is an action as Outline receives it, with the game's
	// status fields.
	InputExample string
	// Status says whether the game has status fields.
	Status bool
	// Dice are the opening's random values, comma separated; empty leaves
	// them out.
	Dice string
}

// renderPrompts fills each prompt's template. A template that does not parse
// or does not run is an error naming the prompt, as an override may carry it.
func renderPrompts(prompts Prompts, data adventurePromptData) (Prompts, error) {
	rendered := Prompts{}
	for name, text := range prompts {
		tmpl, err := template.New(name).Option("missingkey=error").Parse(text)
		if err != nil {
			return nil, fmt.Errorf("prompt %s: %w", name, err)
		}
		var b strings.Builder
		if err := tmpl.Execute(&b, data); err != nil {
			return nil, fmt.Errorf("prompt %s: %w", name, err)
		}
		rendered[name] = b.String()
	}
	return rendered, nil
}

// languageNames are the platform's languages, each named in itself, as the
// platform lists them (go/lang/locales/languages.json).
var languageNames = map[string]string{
	"ar":  "العربية",
	"bar": "Bayrisch",
	"bg":  "Български",
	"bs":  "Bosanski",
	"cs":  "Čeština",
	"da":  "Dansk",
	"de":  "Deutsch",
	"el":  "Ελληνικά",
	"en":  "English",
	"es":  "Español",
	"fa":  "فارسی",
	"fi":  "Suomi",
	"fr":  "Français",
	"hi":  "हिन्दी",
	"hr":  "Hrvatski",
	"hu":  "Magyar",
	"id":  "Bahasa Indonesia",
	"it":  "Italiano",
	"ja":  "日本語",
	"ko":  "한국어",
	"nl":  "Nederlands",
	"no":  "Norsk",
	"pl":  "Polski",
	"ps":  "پښتو",
	"pt":  "Português",
	"ro":  "Română",
	"ru":  "Русский",
	"sk":  "Slovenčina",
	"sl":  "Slovenščina",
	"so":  "Soomaali",
	"sq":  "Shqip",
	"sr":  "Српски",
	"sv":  "Svenska",
	"ti":  "ትግርኛ",
	"tr":  "Türkçe",
	"uk":  "Українська",
	"zh":  "中文",
}

// languageName names a BCP 47 tag's language in itself; a region or script
// subtag falls back to the language, and a language the platform does not list
// is named by its tag.
func languageName(tag string) string {
	if name, known := languageNames[tag]; known {
		return name
	}
	base, _, _ := strings.Cut(tag, "-")
	if name, known := languageNames[base]; known {
		return name
	}
	return tag
}

// inputExample is the example action in game-master, in the shape Outline
// sends: the input, and the status when the game has status fields.
func inputExample(status map[string]string) string {
	example := map[string]any{"input": "drink the potion"}
	if len(status) > 0 {
		example[outlineStatus] = status
	}
	blob, _ := json.Marshal(example)
	return string(blob)
}

// dice joins the opening's random values.
func dice(values []int) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ", ")
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
	properties := map[string]any{
		outlineText:        map[string]any{"type": "string", "description": "Plot outline, just the raw plot - no coloring"},
		outlineImagePrompt: map[string]any{"type": "string", "description": "Vivid description of the scene for image generation"},
	}
	required := []string{outlineText, outlineImagePrompt}
	if len(status) > 0 {
		fields := map[string]any{}
		names := slices.Sorted(maps.Keys(status))
		for _, name := range names {
			fields[name] = text
		}
		properties[outlineStatus] = map[string]any{
			"type":                 "object",
			"description":          "Updated status fields after the action",
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
	prompts, err = renderPrompts(prompts, adventurePromptData{
		Language:     languageName(cfg.Language),
		InputExample: inputExample(cfg.Status),
		Status:       len(cfg.Status) > 0,
		Dice:         dice(cfg.Dice),
	})
	if err != nil {
		return nil, err
	}

	player := blocks.NewPlayerInputText("player-input-text")
	rephrase := blocks.NewToolCall("rephrase", cfg.Tool, prompts[PromptRephrase])
	// Outline takes the thread's first seat, Expand the second.
	story := blocks.NewThread("story-thread", cfg.Thread)
	outline := blocks.NewStructured("outline", story, prompts[PromptGameMaster], OutlineSchema(cfg.Status), outlineResult)
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
