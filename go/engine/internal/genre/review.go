package genre

import (
	"strings"

	"engine/internal/adapters"
)

// ReviewForm is a game's form as an author fills it in, for a review.
type ReviewForm struct {
	Title, Description, Scenario, GameStart, ImageStyle, VoiceStyle, StatusFields string
}

// reviewFields are the fields a review may point at, "none" first: a reply
// with nothing to fix.
var reviewFields = []any{"none", "title", "description", "scenario", "gameStart", "imageStyle", "voiceStyle", "statusFields"}

// reviewPrompt asks for the one point most worth fixing; the genre's own
// words for the fields follow it.
const reviewPrompt = "You coach someone, often a young student, who is writing a game for ChatGameLab, " +
	"where an AI runs the game from what they write. You are given the form as they filled it in. " +
	"Find the single point most worth improving, and only that one: a field done very badly, such as " +
	"a description far too thin to tell players what to expect, or a world with no goal. Judge only " +
	"whether the game can work and players can follow it, never taste, spelling or style. Weigh, in " +
	"this order: the scenario, how the game begins, the description, the image style, then the rest. " +
	"When nothing is seriously wrong, answer field \"none\" and say in one short, friendly sentence " +
	"that it looks good. Otherwise name the field and say in one short, friendly sentence what to " +
	"add or change, speaking to the author as \"you\". Never more than one sentence.\n"

// reviewFieldWords say what each field is for in a genre.
func reviewFieldWords(g string) string {
	common := "- title: the game's name in the gallery.\n" +
		"- description: one or two sentences in the gallery, so players know what to expect.\n" +
		"- imageStyle: how the game's pictures look.\n" +
		"- voiceStyle: how the game's voice sounds.\n"
	if g == "npc-live" {
		return "The game is a Character: the player talks by voice with one character.\n" + common +
			"- scenario: who the character is: personality, what they know, how they talk, and what they will never give in on.\n" +
			"- gameStart: the situation when the conversation starts, from which the character's first words are written.\n"
	}
	if g == "npc-duo" {
		return "The game is a Discussion: two characters talk with each other by voice, and the player may join in or only listen.\n" + common +
			"- scenario: who the two characters are, the first and the second, each with personality, what they want and how they talk; they should have something to talk or argue about.\n" +
			"- gameStart: the scene both are in when it begins, from which the first character's first words are written.\n"
	}
	return "The game is an Adventure: the AI narrates a world the player explores, with pictures and values it tracks.\n" + common +
		"- scenario: the world, the player's role, the goal, and how the player knows they have won.\n" +
		"- gameStart: the first scene, and how the player knows what to do.\n" +
		"- statusFields: the values the game tracks, as JSON; few beat many.\n"
}

// Review is the call that reviews a form: the system prompt, the message
// and the reply's schema, {field, text}.
func Review(genre string, form ReviewForm, language string) (system, message string, schema *adapters.Schema) {
	system = reviewPrompt + reviewFieldWords(genre)
	if language != "" {
		system += "\nWrite the sentence in " + languageName(language) + "."
	}
	var b strings.Builder
	field := func(name, value string) {
		b.WriteString(name + ":\n")
		if strings.TrimSpace(value) == "" {
			b.WriteString("(empty)\n\n")
			return
		}
		b.WriteString(value + "\n\n")
	}
	field("title", form.Title)
	field("description", form.Description)
	field("scenario", form.Scenario)
	field("gameStart", form.GameStart)
	field("imageStyle", form.ImageStyle)
	field("voiceStyle", form.VoiceStyle)
	if genre != "npc-live" && genre != "npc-duo" {
		field("statusFields", form.StatusFields)
	}
	return system, b.String(), &adapters.Schema{Name: "review", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"field": map[string]any{"type": "string", "enum": reviewFields, "description": "the field the point is about, or none"},
			"text":  map[string]any{"type": "string", "description": "one short sentence"},
		},
		"required":             []any{"field", "text"},
		"additionalProperties": false,
	}}
}
