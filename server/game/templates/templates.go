package templates

import (
	"cgl/functional"
	"cgl/game/status"
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
	// PromptNarratePlotOutline is sent after each JSON response to get prose narration
	PromptNarratePlotOutline = "NARRATE the summary into prose. STRICT RULES: 1-3 sentences MAXIMUM. No headers, no markdown, no lists. Do NOT repeat status fields. End on an open note. Be brief and atmospheric."
)

func ImageStyleOrDefault(style string) string {
	if style == "" {
		return DefaultImageStyle
	}
	return style
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
2. I ask you to NARRATE → You respond with plain text prose (1-3 sentences MAXIMUM, be brief)

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

Rules for Phase 2:
- 1-3 short sentences maximum
- No headers, no markdown, no lists
- Describe the scene, not a story structure
- DON'T repeat the status fields, only write the story content (status fields are reported in Phase 1 only!)
- End on an open note - let the player decide what to do next

---
NARRATIVE STYLE
---
- Follow the scenario's defined language and literary style
- Write like a skilled dungeon master: brief, atmospheric, action-focused
- Stay in character as the game world at all times

The scenario:
{{SCENARIO}}
{{GAME_START}}`

func GetTemplate(game *obj.Game) (string, error) {
	var statusFields []obj.StatusField
	if game.StatusFields != "" {
		if err := json.Unmarshal([]byte(game.StatusFields), &statusFields); err != nil {
			return "", fmt.Errorf("failed to unmarshal status fields while parsing game: %w", err)
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
		Message:     "You drink the potion. You feel a little bit dizzy. You feel a little bit stronger.",
		Status:      statusMap,
		ImagePrompt: functional.Ptr("a castle in the background, green grass, late afternoon"),
	}
	actionOutputStr, _ := json.Marshal(actionOutput)

	instructions := systemTemplate
	instructions = strings.ReplaceAll(instructions, "{{INPUT_EXAMPLE}}", string(actionInputStr))
	instructions = strings.ReplaceAll(instructions, "{{OUTPUT_EXAMPLE}}", string(actionOutputStr))
	instructions = strings.ReplaceAll(instructions, "{{TYPE_PLAYER}}", obj.GameSessionMessageTypePlayer)
	instructions = strings.ReplaceAll(instructions, "{{TYPE_SYSTEM}}", obj.GameSessionMessageTypeSystem)
	instructions = strings.ReplaceAll(instructions, "{{SCENARIO}}", game.SystemMessageScenario)

	// Append game start instructions if provided by the game creator
	if game.SystemMessageGameStart != "" {
		instructions = strings.ReplaceAll(instructions, "{{GAME_START}}",
			fmt.Sprintf("\nHow to start the game:\n%s", game.SystemMessageGameStart))
	}

	return instructions, nil
}
