package game

import (
	"cgl/log"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cgl/game/ai"
)

// ThemeGenerationPrompt is the system prompt for generating game themes
const ThemeGenerationPrompt = `You are a visual theme generator for a text adventure game. Generate a JSON theme based on the game's name, description, and scenario.

RULES:
1. Only customize what CLEARLY fits the game's theme. Default is better than over-styling.
2. Include ALL fields. Do not omit any.
3. Output ONLY valid JSON. No explanation, no markdown.

OPTIONS:

corners.style: "none" (default) | "brackets" (tech/sci-fi) | "flourish" (fantasy) | "arrows" (exploration) | "dots" (mystery)
corners.color: "amber" (default) | "emerald" (nature) | "cyan" (tech) | "violet" (magic) | "rose" (romance/horror) | "slate" (noir)

background.animation: "none" (default) | "stars" (space) | "rain" (noir) | "fog" (horror) | "particles" (magic) | "scanlines" (retro)
background.tint: "neutral" (default) | "warm" (cozy) | "cool" (tech/cold) | "dark" (horror/noir)

player.color: Same colors as corners. Default: "cyan"
player.indicator: "none" (default) | "dot" | "arrow" | "chevron" | "diamond"
player.showChevron: false (default). Only true for terminal/hacker themes.
player.bgColor: "cyan" (default) | "amber" | "violet" | "slate" | "white" | "emerald" | "rose"

gameMessage.dropCap: true (default). Set false for terminal/mono/minimal styles.
gameMessage.dropCapColor: Same colors as corners. Default: matches corners.color.

thinking.text: Thematic loading phrase IN THE SAME LANGUAGE as the scenario. Default: "The story unfolds..."
  Examples: "Decrypting transmission...", "The mist clears...", "Die Geschichte entfaltet sich...", "El oráculo responde..."
thinking.style: "dots" (default) | "spinner" | "pulse" | "typewriter"

typography.messages: "sans" (default) | "serif" (classic/fantasy) | "mono" (terminal) | "fantasy" (high fantasy only)

statusEmojis: Map status field names to emoji. Use {} if no obvious mappings.
  Common: Health→❤️, Gold→🪙, Energy→⚡, Mana→🔮, Food→🍖, Time→⏰

EXAMPLE (mostly defaults for a generic adventure):
{
  "corners": { "style": "none", "color": "amber" },
  "background": { "animation": "none", "tint": "neutral" },
  "player": { "color": "cyan", "indicator": "none", "showChevron": false, "bgColor": "cyan" },
  "gameMessage": { "dropCap": true, "dropCapColor": "amber" },
  "thinking": { "text": "The story unfolds...", "style": "dots" },
  "typography": { "messages": "sans" },
  "statusEmojis": {}
}`

// GenerateTheme generates a visual theme for the game based on its description
func GenerateTheme(ctx context.Context, session *obj.GameSession, game *obj.Game) (*obj.GameTheme, error) {
	if session == nil || session.ApiKey == nil {
		return nil, fmt.Errorf("session or API key is nil")
	}

	log.Debug("generating theme for game", "game_id", game.ID, "game_name", game.Name)

	// Get AI platform
	platform, _, err := ai.GetAiPlatform(session.AiPlatform, session.AiModel)
	if err != nil {
		log.Debug("failed to get AI platform for theme generation", "error", err)
		return nil, fmt.Errorf("failed to get AI platform: %w", err)
	}

	// Build the user prompt with game details
	scenario := game.SystemMessageScenario
	if scenario == "" {
		scenario = "(No scenario defined)"
	}

	statusFields := game.StatusFields
	if statusFields == "" {
		statusFields = "(No status fields defined)"
	}

	userPrompt := fmt.Sprintf(`Generate a visual theme for this game. Use neutral defaults unless the game clearly suggests a specific style.

GAME NAME: %s

DESCRIPTION: %s

SCENARIO/SETTING:
%s

STATUS FIELDS (for emoji mapping - only add emoji if it fits):
%s

Generate the JSON theme. Remember: use defaults for most options, only customize what clearly fits the theme.`,
		game.Name,
		game.Description,
		truncateString(scenario, 800),
		statusFields,
	)

	// Call AI to generate theme
	log.Debug("calling AI to generate theme")
	response, err := platform.GenerateTheme(ctx, session, ThemeGenerationPrompt, userPrompt)
	if err != nil {
		log.Debug("AI theme generation failed", "error", err)
		return nil, fmt.Errorf("failed to generate theme: %w", err)
	}

	// Parse the JSON response
	theme, err := parseThemeResponse(response)
	if err != nil {
		log.Debug("failed to parse theme response", "error", err, "response", response)
		// Return default theme on parse error
		return defaultTheme(), nil
	}

	log.Debug("theme generated successfully", "theme", theme)
	return theme, nil
}

// parseThemeResponse parses the AI response into a GameTheme
func parseThemeResponse(response string) (*obj.GameTheme, error) {
	// Clean up response - remove markdown code blocks if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var theme obj.GameTheme
	if err := json.Unmarshal([]byte(response), &theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme JSON: %w", err)
	}

	// Validate and set defaults for missing/invalid values
	theme = validateTheme(theme)

	return &theme, nil
}

// validateTheme ensures all theme values are valid, setting defaults for invalid ones
func validateTheme(theme obj.GameTheme) obj.GameTheme {
	// Validate corner style (default: none)
	validStyles := map[string]bool{"none": true, "brackets": true, "flourish": true, "arrows": true, "dots": true}
	if !validStyles[theme.Corners.Style] {
		theme.Corners.Style = "none"
	}

	// Validate colors (default: amber for corners, cyan for player)
	validColors := map[string]bool{"amber": true, "emerald": true, "cyan": true, "violet": true, "rose": true, "slate": true}
	if !validColors[theme.Corners.Color] {
		theme.Corners.Color = "amber"
	}
	if !validColors[theme.Player.Color] {
		theme.Player.Color = "cyan"
	}
	if !validColors[theme.GameMessage.DropCapColor] {
		// Default drop cap color should align with the overall accent (corners)
		theme.GameMessage.DropCapColor = theme.Corners.Color
	}

	// Validate background animation (default: none)
	validAnimations := map[string]bool{"none": true, "stars": true, "rain": true, "fog": true, "particles": true, "scanlines": true}
	if !validAnimations[theme.Background.Animation] {
		theme.Background.Animation = "none"
	}

	// Validate background tint (default: neutral)
	validTints := map[string]bool{"neutral": true, "warm": true, "cool": true, "dark": true}
	if !validTints[theme.Background.Tint] {
		theme.Background.Tint = "neutral"
	}

	// Validate player indicator (default: none)
	validIndicators := map[string]bool{"none": true, "dot": true, "arrow": true, "chevron": true, "diamond": true}
	if !validIndicators[theme.Player.Indicator] {
		theme.Player.Indicator = "none"
	}

	// Validate player bg color (default: cyan)
	validPlayerBgColors := map[string]bool{"cyan": true, "amber": true, "violet": true, "slate": true, "white": true, "emerald": true, "rose": true}
	if !validPlayerBgColors[theme.Player.BgColor] {
		theme.Player.BgColor = "cyan"
	}

	// Validate thinking style
	validThinkingStyles := map[string]bool{"dots": true, "spinner": true, "pulse": true, "typewriter": true}
	if !validThinkingStyles[theme.Thinking.Style] {
		theme.Thinking.Style = "dots"
	}
	if theme.Thinking.Text == "" {
		theme.Thinking.Text = "The story unfolds..."
	}

	// Validate typography
	validFonts := map[string]bool{"serif": true, "sans": true, "mono": true, "fantasy": true}
	if !validFonts[theme.Typography.Messages] {
		theme.Typography.Messages = "sans"
	}

	// Initialize status emojis if nil
	if theme.StatusEmojis == nil {
		theme.StatusEmojis = make(map[string]string)
	}

	return theme
}

// defaultTheme returns the default neutral theme
func defaultTheme() *obj.GameTheme {
	return &obj.GameTheme{
		Corners: obj.GameThemeCorners{
			Style: "none",
			Color: "amber",
		},
		Background: obj.GameThemeBackground{
			Animation: "none",
			Tint:      "neutral",
		},
		Player: obj.GameThemePlayer{
			Color:       "cyan",
			Indicator:   "none",
			ShowChevron: false,
			BgColor:     "cyan",
		},
		GameMessage: obj.GameThemeGameMessage{
			DropCap:      true,
			DropCapColor: "amber",
		},
		Thinking: obj.GameThemeThinking{
			Text:  "The story unfolds...",
			Style: "dots",
		},
		Typography: obj.GameThemeTypography{
			Messages: "sans",
		},
		StatusEmojis: make(map[string]string),
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
