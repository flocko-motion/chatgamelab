// Package genre holds one wiring per genre, as Go code so the compiler checks each graph.
package genre

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// AdventureConfig is what the wiring needs from a session spec, already resolved.
type AdventureConfig struct {
	Tool adapters.Tool
	// Hearing takes the player's speech.
	Hearing adapters.Hearing
	// Thread runs the story model's conversation.
	Thread adapters.Thread
	Image  adapters.Image

	// Scenario and Guardrail stay apart: nothing a designer writes shares a
	// message with the youth-protection rule where a provider can keep them
	// apart.
	Scenario  string
	Guardrail string

	// Description is the game's idea in a sentence, for the picture prompt.
	Description string

	// ImageStyle is the designer's look; empty takes AdventureImageStyle.
	ImageStyle string

	// StartingMessage is how the game starts, in the designer's words: part
	// of the System Message, left out when empty.
	StartingMessage string

	// Status seeds the values the game keeps, such as Health.
	Status map[string]string

	// Language is the player's, as a BCP 47 tag; empty leaves it to the model.
	Language string

	// Dice are the random values the opening bases its choices on, rolled
	// per game; none leaves them out.
	Dice []int

	// PromptOverride retunes the genre's prompts by name, without a new build.
	PromptOverride map[string]string

	// ConstraintFrequency sets, by block name, how often a threaded block's
	// calls carry the constraints: 1, the default, is every call, up to 10.
	ConstraintFrequency map[string]int
}

// Prompt names are constants so a rename fails to compile.
const (
	PromptGameMaster       = "game-master"
	PromptRephrase         = "rephrase"
	PromptRephraseAudio    = "rephrase-audio"
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
	// The game engine prompt, the head of the story thread's System Message,
	// sent once as the thread's first message. The input example is filled in
	// with the game's own status fields.
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

	// Rephrase for spoken input: the model hears the player and answers in
	// SpokenSchema, their words as they said them and the action rephrased.
	PromptRephraseAudio: "You hear the player of a text adventure say what they do next. Reply with two fields.\n" +
		`"heard": a transcript of the player's words, word for word, in the language they spoke. ` +
		`Do not translate it, rephrase it or change its person: if the player says "I", write "I".` +
		`{{with .Language}} The player most likely speaks {{.}}; write down whatever language they speak.{{end}}` + "\n" +
		`"action": what the player said, rephrased in third person, making the outcome uncertain.` + "\n" +
		"If you hear no words, leave both fields empty.\n" +
		"Example: the player says 'I climb the old tower and jump onto the roof' → " +
		`{"heard": "I climb the old tower and jump onto the roof", ` +
		`"action": "The player climbs the old tower, hoping to jump onto the roof."}` +
		`{{with .Language}}` + "\n" + `Write "action" in {{.}}.{{end}}`,

	// The Initialization Prompt: Outline's message on the first turn, sent by
	// the gate as the game's own message. The status sentence only when the
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

	// The Expand Prompt: Expand's message on every turn after the first.
	// The outline is on the thread already, as the reply before.
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

// The fields of Rephrase's reply to speech.
const (
	spokenHeard  = "heard"
	spokenAction = "action"
)

// spokenResult feeds the action to Outline and what was heard to the player's line.
var spokenResult = blocks.Result{Text: spokenAction, SecondaryText: spokenHeard}

// SpokenSchema is the shape of Rephrase's reply to speech, a call parameter of
// the genre's: the player's words verbatim, in their language, and the action,
// in third person with the outcome uncertain.
func SpokenSchema() adapters.Schema {
	return adapters.Schema{Name: "spoken", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			spokenHeard:  map[string]any{"type": "string", "description": "The player's words verbatim, in the language they spoke"},
			spokenAction: map[string]any{"type": "string", "description": "The action in third person, its outcome uncertain"},
		},
		"required":             []string{spokenHeard, spokenAction},
		"additionalProperties": false,
	}}
}

// The length a clip of speech may have: a press shorter than the shortest
// sends nothing, and a recording stops at the longest. The engine takes a
// clip up to clipSlack past the longest, the frames a page records while it
// stops.
const (
	ShortestClip = 300 * time.Millisecond
	LongestClip  = 30 * time.Second
	clipSlack    = time.Second
)

// The threaded blocks, whose constraint frequency a spec may set by name.
const (
	BlockOutline = "outline"
	BlockExpand  = "expand"
)

// AdventureConstrained is every block whose calls carry the constraints.
func AdventureConstrained() []string { return []string{BlockExpand, BlockOutline} }

// constraintsLead introduces the youth-protection rule, wherever it goes.
const constraintsLead = "NARRATION RULES must be respected: "

// worded is the youth-protection rule as a message carries it; none is empty.
func worded(guardrail string) string {
	if strings.TrimSpace(guardrail) == "" {
		return ""
	}
	return constraintsLead + guardrail
}

// systemMessage is the story thread's System Message: the game engine prompt,
// the scenario, how the game starts when the designer says so, and the
// constraints.
func systemMessage(gameMaster, scenario, start, constraints string) string {
	parts := []string{gameMaster, "The scenario:\n" + scenario}
	if strings.TrimSpace(start) != "" {
		parts = append(parts, "How to start the game:\n"+start)
	}
	if constraints != "" {
		parts = append(parts, constraints)
	}
	return strings.Join(parts, "\n\n")
}

// longScenario is the length past which a scenario is condensed for the
// picture prompts.
const longScenario = 500

// The props preparation hands the picture prompt.
const (
	partScenario = "scenario"
	partStyle    = "style"
)

// imagePrompt builds a picture prompt from one reply of Outline's, in the
// order the image model reads best: the scene frame's first line, the game's
// idea and scenario, the scene and what the picture shows, the style, and the
// frame's last line. The scenario is the condensed one when there is one, the
// style the adapted one, otherwise AdventureImageStyle.
func imagePrompt(scene, description, scenario string) func(blocks.Parts) string {
	lines := strings.Split(strings.TrimSpace(scene), "\n")
	first, last := lines[0], lines[len(lines)-1]
	return func(p blocks.Parts) string {
		parts := []string{first}
		if description != "" {
			parts = append(parts, "The game idea is: "+description)
		}
		setting := scenario
		if condensed := p.Props[partScenario]; condensed != "" {
			setting = condensed
		}
		if setting != "" {
			parts = append(parts, "The game scenario is: "+setting)
		}
		if p.Secondary != "" {
			parts = append(parts, "The current scene is: "+p.Secondary)
		}
		if p.Text != "" {
			parts = append(parts, "The visual should show: "+p.Text)
		}
		style := p.Props[partStyle]
		if style == "" {
			style = AdventureImageStyle
		}
		parts = append(parts, "The artistic style should be: "+style)
		if len(lines) > 1 {
			parts = append(parts, last)
		}
		return strings.Join(parts, "\n")
	}
}

// styleMessage is the image-style preparation's message: the style, and the
// youth-protection rule it is adapted to.
func styleMessage(style, guardrail string) string {
	if strings.TrimSpace(style) == "" {
		style = AdventureImageStyle
	}
	message := "Image Style: " + style
	if strings.TrimSpace(guardrail) != "" {
		message += "\n\nWorkshop Constraints:\n" + guardrail
	}
	return message
}

// The codes the genres' error sinks report, which the engine names as its
// error codes.
const (
	CodeTurnFailed  = "turn_failed"
	CodeImageFailed = "image_failed"
	// CodeVoiceFailed: nothing came through of what the player said.
	CodeVoiceFailed = "voice_failed"
)

// ErrVoiceOff refuses speech while voice input is off.
var ErrVoiceOff = errors.New("voice input is off")

// NewAdventure is the turn-based genre. Preparation condenses a long scenario
// and adapts the image style; then the world opens the game: the gate sends
// the Initialization Prompt to Outline. Each turn after, Rephrase turns the
// player's action into third person, Outline decides the plot, image prompt
// and status in one structured call, Expand writes the prose on the same
// thread, and the turn fans out to media. While narration is on, Expand's
// turns are spoken: the prose's transcript and its audio reach the player as
// they come. The player's input is released once the opening's prose is done.
//
// The player types or speaks. A typed line goes to Rephrase and, at once, to
// the player's line on the `said` stream; a spoken clip goes to Rephrase's
// spoken twin, on the hearing model, which answers what it heard, for the
// player's line, and the action, for Outline. Voice input switches the spoken
// twin, and nothing coming through of a clip is the `voice_failed` error.
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

	constraints := worded(cfg.Guardrail)
	player := blocks.NewPlayerInputText("player-input-text")
	rephrase := blocks.NewToolCall("rephrase", cfg.Tool, prompts[PromptRephrase])
	// Like Rephrase, it carries no youth-protection rule: it only restates.
	speaker := blocks.NewPlayerInputClips("player-input-audio")
	hearing := blocks.NewHearingCall("rephrase-audio", cfg.Hearing, prompts[PromptRephraseAudio], SpokenSchema(), spokenResult)
	// Outline takes the thread's first seat, Expand the second.
	story := blocks.NewThread("story-thread", cfg.Thread,
		systemMessage(prompts[PromptGameMaster], cfg.Scenario, cfg.StartingMessage, constraints), constraints)
	outline := blocks.NewStructured(BlockOutline, story, OutlineSchema(cfg.Status), outlineResult)
	expand := blocks.NewThreaded(BlockExpand, story, blocks.Prompt{First: prompts[PromptOpeningExpand], Then: prompts[PromptExpand]})
	expand.Clean = cleanNarration
	constrained := map[string]blocks.Constrained{outline.NodeName(): outline, expand.NodeName(): expand}

	picturePrompt := blocks.NewCompose("image-prompt", imagePrompt(prompts[PromptScene], cfg.Description, cfg.Scenario))
	image := blocks.NewImage("image", cfg.Image)

	outText := blocks.NewPlayerOutputStream("out-text", "text")
	outSaid := blocks.NewPlayerOutputText("out-said", "said")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")
	outProps := blocks.NewPlayerOutputProps("out-props", "props")
	props := blocks.NewPropsStore("props-store", cfg.Status)
	turnFailed := blocks.NewErrorSink("turn-failed", CodeTurnFailed)
	imageFailed := blocks.NewErrorSink("image-failed", CodeImageFailed)
	voiceFailed := blocks.NewErrorSink("voice-failed", CodeVoiceFailed)
	problem := blocks.NewProblemSink("problem")

	// Preparation: each keep asks its tool call once. A short scenario needs
	// no condensing, so its keep asks nothing.
	condenseRequest := ""
	if len([]rune(cfg.Scenario)) > longScenario {
		condenseRequest = "Scenario:\n" + cfg.Scenario
	}
	condenser := blocks.NewToolCall("condense-scenario", cfg.Tool, prompts[PromptCondenseScenario])
	condensed := blocks.NewKeep("condensed-scenario", partScenario, condenseRequest)
	adapter := blocks.NewToolCall("image-style", cfg.Tool, prompts[PromptImageStyle])
	adapted := blocks.NewKeep("adapted-style", partStyle, styleMessage(cfg.ImageStyle, cfg.Guardrail))

	// The world has the initiative: once preparation is done, the gate sends
	// the Initialization Prompt.
	gate := blocks.NewGate("start-game", prompts[PromptOpening])

	g := ports.NewGraph("adventure")
	// The thread is exported once, under its own name, for both its blocks.
	g.Share(story)

	g.ConnectTextOut(condensed, condenser)
	g.ConnectTextOut(condenser, condensed)
	g.ConnectTextOut(adapted, adapter)
	g.ConnectTextOut(adapter, adapted)
	// Both are tool calls before the game, so the game cannot start without them.
	g.ConnectFailure(condenser, problem)
	g.ConnectFailure(adapter, problem)
	g.ConnectState(condensed, gate)
	g.ConnectState(adapted, gate)
	g.ConnectPropsOut(condensed, picturePrompt)
	g.ConnectPropsOut(adapted, picturePrompt)

	// The opening skips Rephrase: it is the game's own message, not an action.
	g.ConnectOpening(gate, outline)
	g.ConnectTextOut(player, rephrase)
	g.ConnectTextOut(rephrase, outline)
	g.ConnectFailure(rephrase, turnFailed)

	// The player's line: typed at once, spoken once heard.
	g.ConnectTextOut(player, outSaid)
	g.ConnectAudioOut(speaker, hearing)
	g.ConnectTextOut(hearing, outline)
	g.ConnectSecondaryTextOut(hearing, outSaid)
	g.ConnectFailure(hearing, voiceFailed)

	g.ConnectTextOut(outline, expand)
	// The picture prompt pairs one reply's image prompt with its outline.
	g.ConnectSecondaryTextOut(outline, picturePrompt)
	g.ConnectTextOutToSecondary(outline, picturePrompt)
	// The store holds status values and is the one place to seed or correct them.
	g.ConnectPropsOut(outline, props)
	g.ConnectPropsOut(props, outline)
	g.ConnectPropsOut(props, outProps)
	g.ConnectFailure(outline, turnFailed)

	// The player watches the prose appear, and hears it as it is spoken.
	g.ConnectStreamOut(expand, outText)
	g.ConnectAudioOut(expand, outAudio)
	g.ConnectFailure(expand, turnFailed)

	g.ConnectTextOut(picturePrompt, image)
	g.ConnectImageOut(image, outImage)
	// A picture that fails costs the picture; the turn stands.
	g.ConnectFailure(image, imageFailed)

	// The player acts once the opening's prose is done, or once it failed.
	g.ConnectState(outline, player)
	g.ConnectState(expand, player)
	g.ConnectState(outline, speaker)
	g.ConnectState(expand, speaker)

	w := &Wiring{
		Graph:       g,
		Gate:        gate,
		Prompts:     prompts,
		Constrained: constrained,
		// The graph's shape is fixed; a toggle that is off stops its block
		// working. Narration has Expand's turns spoken, and the session runs
		// the story thread on the tier's speaking model meanwhile.
		Switches: map[adapters.Toggle][]ports.Switchable{
			adapters.ToggleImages:   {image},
			adapters.ToggleAudioOut: {expand.Narration()},
			adapters.ToggleAudioIn:  {hearing},
		},
		Say:   player.Say,
		Speak: speak(hearing, speaker),
		// The player holds a button to talk, or types.
		Inputs: []ports.InputMode{ports.InputText, ports.InputAudioPushToTalk},
		// A picture per turn, each belonging to the scene it illustrates.
		Imagery: ports.ImageryPerTurn,
	}
	for _, block := range slices.Sorted(maps.Keys(cfg.ConstraintFrequency)) {
		if err := w.SetConstraintFrequency(block, cfg.ConstraintFrequency[block]); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// speak takes one clip of the player's speech, as one utterance, while voice
// input is on; a clip too short to hold words, or longer than a recording
// runs, is refused.
func speak(hearing *blocks.HearingCall, speaker *blocks.PlayerInputAudio) func([]byte) error {
	return func(clip []byte) error {
		if !hearing.Active() {
			return ErrVoiceOff
		}
		length := time.Duration(adapters.Clip(clip).Seconds() * float64(time.Second))
		if length < ShortestClip || length > LongestClip+clipSlack {
			return fmt.Errorf("a clip of %s: want %s to %s", length.Round(time.Millisecond), ShortestClip, LongestClip)
		}
		speaker.Speak(clip)
		return nil
	}
}
