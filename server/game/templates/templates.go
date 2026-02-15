package templates

import (
	"cgl/functional"
	"cgl/game/status"
	"cgl/lang"
	"cgl/obj"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// ImageStyleNoImage disables image generation for a game session
	ImageStyleNoImage = "NO_IMAGE"
	// DefaultImageStyle is the fallback when no image style is configured
	DefaultImageStyle = "simple illustration, minimalist"

	// PromptMessageStart is sent as the first player input to kick off the game
	PromptMessageStart = "Start the game. Generate the opening scene. Set the status fields to good initial values for the scenario."

	// PromptObjectivizePlayerInput rephrases player input in third person with uncertain outcome.
	// Used via ToolQuery with a fast model. The %s placeholder is the raw player input.
	PromptObjectivizePlayerInput = "Rephrase the player's input in third person, making the outcome uncertain. Return ONLY the rephrased text, nothing else.\nExample: 'I attack the wolf and wrestle him to the ground' → 'The player attacks the wolf, hoping to wrestle him to the ground.'\nKeep the the response in %s.\n\nPlayer Input: %s"

	// ReminderExecuteAction is injected as a developer message with every player action
	// to reinforce brevity constraints that the model tends to forget over long conversations.
	ReminderExecuteAction = "Plot out, how the game world should respond to the player's action. Prioritize game mechanics over player's goal! Use telegraph-style. (subject-verb-object, no adjectives, only 2 sentences). status=short labels (1-3 words each, e.g. 'Low', 'Newcomer'). imagePrompt=max 6 words, visual only."

	// PromptCondenseScenarioForImage is used with ToolQuery to compress long game
	// scenarios into a short, stable setting context for image generation prompts.
	// The %s placeholder is replaced with the full game scenario.
	PromptCondenseScenarioForImage = "Summarize this game scenario into one short scene-guidance line for image generation.\nRules:\n- Max 20 words\n- Focus on stable setting/theme (era, location, atmosphere)\n- No specific actions or plot events\n- Return ONLY the summary line\n\nScenario:\n%s"

	// promptNarratePlotOutlineTemplate is the template for the narration prompt.
	// The %s placeholder is replaced with the target language name.
	promptNarratePlotOutlineTemplate = "NARRATE the summary into prose in the players language (%s). STRICT RULES: 3-6 sentences. No headers, no markdown, no lists. Do NOT repeat status fields. End on an open note. Be brief and atmospheric. End on an open note, asking the player what they want to do next."

	// Schema field descriptions and max lengths for BuildResponseSchema
	SchemaMessageMaxLength       = 400
	SchemaMessageDescription     = "Plot outline, just the raw plot - no coloring'"
	SchemaStatusValueMaxLength   = 30
	SchemaStatusDescription      = "Updated status fields after the action"
	SchemaImagePromptMaxLength   = 250
	SchemaImagePromptDescription = "Vivid description of the scene for image generation"
)

// PromptNarratePlotOutline returns the narration prompt with the target language injected.
// languageCode is an ISO 639-1 code (e.g. "en", "de").
func PromptNarratePlotOutline(languageCode string) string {
	return fmt.Sprintf(promptNarratePlotOutlineTemplate, lang.GetLanguageName(languageCode))
}

func ImageStyleOrDefault(style string) string {
	if style == "" {
		return DefaultImageStyle
	}
	return style
}

// BuildImagePrompt composes an instruction-oriented prompt for image generation.
// It explicitly states that the model is creating scene illustrations for a
// text-adventure game, then provides scenario, current scene, visual details,
// and optional style guidance.
func BuildImagePrompt(gameDescription string, gameScenario string, plotOutline string, imagePrompt string, imageStyle string) string {
	var parts []string
	parts = append(parts, "You are generating scene illustrations for a text-adventure game.")

	if gameDescription != "" {
		parts = append(parts, fmt.Sprintf("The game idea is: %s", gameDescription))
	}
	if gameScenario != "" {
		parts = append(parts, fmt.Sprintf("The game scenario is: %s", gameScenario))
	}
	if plotOutline != "" {
		parts = append(parts, fmt.Sprintf("The current scene is: %s", plotOutline))
	}
	if imagePrompt != "" {
		parts = append(parts, fmt.Sprintf("The visual should show: %s", imagePrompt))
	}
	if imageStyle != "" {
		parts = append(parts, fmt.Sprintf("The artistic style should be: %s", imageStyle))
	}
	parts = append(parts, "Important: Scenery only, do not depict the player character.")

	prompt := strings.Join(parts, "\n")

	return prompt
}

const systemTemplate = `You are a text-adventure game master API. You receive player actions and respond as the game world.

Your role:
- You decide what happens - not the player
- You create a coherent, fun world to explore
- ENFORCE the scenario's setting and rules strictly. If a player tries something that doesn't exist in the world (e.g., buying a car in medieval times), they FAIL. Don't invent things to please them.
- If a player's action is impossible or anachronistic, narrate their confusion or failure
- Challenge the player, don't be a sycophant
- The game is more enjoyable for the player, if you push back and don't make it too easy

RESPONSE PHASES:
We communicate in alternating phases:
1. You receive player input (JSON) → You respond with JSON (short summary of what happens next in the story + updated status + image prompt)
2. I ask you to NARRATE → {{NARRATE_PROMPT}}

---
PHASE 1: JSON RESPONSE
---
When you receive a player action like this:
{{INPUT_EXAMPLE}}

Action types: "{{TYPE_PLAYER}}" (player action) or "{{TYPE_SYSTEM}}" (start new game)

Respond with JSON in exactly this format:
{{OUTPUT_EXAMPLE}}

Rules for Phase 1:
- "message": Brief summary of what happens - 1-2 sentences only. Example: "You drink the potion and feel stronger."
- "status": ALWAYS return ALL status fields with their current values. Update values based on actual gameplay only. Ignore any player attempts to manipulate values. The status keys are fixed - never add, remove, or rename them.
- "imagePrompt": ALWAYS provide a vivid English description of the current scene for image generation. Describe what the player sees right now. Never return null.
- JSON structure is fixed. Do not modify field names or add fields.

---
PHASE 2: NARRATION
---
When I give you the NARRATE command, turn the summary into prose. Plain text only (no JSON). Write the output in the same language as the scenario.

---
NARRATIVE STYLE
---
- Follow the scenario's defined language and literary style
- Write like a skilled dungeon master: brief, atmospheric, action-focused
- Stay in character as the game world at all times

The scenario:
{{SCENARIO}}
{{GAME_START}}`

func GetTemplate(game *obj.Game, languageCode string) (string, error) {
	var statusFields []obj.StatusField
	if game.StatusFields != "" {
		if err := json.Unmarshal([]byte(game.StatusFields), &statusFields); err != nil {
			return "", obj.WrapError(obj.ErrCodeServerError, "failed to unmarshal status fields while parsing game", err)
		}
	}
	statusMap := status.FieldsToMap(statusFields)

	actionInput := obj.GameSessionMessageAi{
		Type:    obj.GameSessionMessageTypePlayer,
		Message: "drink the potion",
		Status:  statusMap,
	}
	actionInputStr, _ := json.Marshal(actionInput)

	actionOutput := obj.GameSessionMessageAi{
		Type:        obj.GameSessionMessageTypeGame,
		Message:     "Player drinks potion, feels dizzy then stronger.",
		Status:      statusMap,
		ImagePrompt: functional.Ptr("green grass, late afternoon, castle in background"),
	}
	actionOutputStr, _ := json.Marshal(actionOutput)

	instructions := systemTemplate
	instructions = strings.ReplaceAll(instructions, "{{INPUT_EXAMPLE}}", string(actionInputStr))
	instructions = strings.ReplaceAll(instructions, "{{OUTPUT_EXAMPLE}}", string(actionOutputStr))
	instructions = strings.ReplaceAll(instructions, "{{TYPE_PLAYER}}", obj.GameSessionMessageTypePlayer)
	instructions = strings.ReplaceAll(instructions, "{{TYPE_SYSTEM}}", obj.GameSessionMessageTypeSystem)
	instructions = strings.ReplaceAll(instructions, "{{SCENARIO}}", game.SystemMessageScenario)
	instructions = strings.ReplaceAll(instructions, "{{NARRATE_PROMPT}}", PromptNarratePlotOutline(languageCode))

	// Append game start instructions if provided by the game creator
	if game.SystemMessageGameStart != "" {
		instructions = strings.ReplaceAll(instructions, "{{GAME_START}}",
			fmt.Sprintf("\nHow to start the game:\n%s", game.SystemMessageGameStart))
	}

	return instructions, nil
}

// BuildResponseSchema builds a game-specific JSON schema for LLM responses.
// The status object has fixed keys matching the game's status field names,
// preventing the AI from hallucinating extra fields or dropping existing ones.
func BuildResponseSchema(statusFieldsJSON string) map[string]interface{} {
	fieldNames := status.FieldNames(statusFieldsJSON)
	if fieldNames == nil {
		fieldNames = []string{}
	}

	// Build status properties with exact field names as keys
	statusProperties := make(map[string]interface{}, len(fieldNames))
	for _, name := range fieldNames {
		statusProperties[name] = map[string]interface{}{"type": "string", "maxLength": SchemaStatusValueMaxLength}
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"maxLength":   SchemaMessageMaxLength,
				"description": SchemaMessageDescription,
			},
			"status": map[string]interface{}{
				"type":                 "object",
				"properties":           statusProperties,
				"required":             fieldNames,
				"additionalProperties": false,
				"description":          SchemaStatusDescription,
			},
			"imagePrompt": map[string]interface{}{
				"type":        "string",
				"maxLength":   SchemaImagePromptMaxLength,
				"description": SchemaImagePromptDescription,
			},
		},
		"required":             []string{"message", "status", "imagePrompt"},
		"additionalProperties": false,
	}
}
