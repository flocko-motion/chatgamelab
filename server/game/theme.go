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
const ThemeGenerationPrompt = `You are a visual theme generator for a text adventure game player interface.
Based on the game's name, description, scenario, and status fields, generate a JSON theme configuration.

IMPORTANT: Only customize settings that fit the game's theme. Use neutral/default values for anything that doesn't have a strong thematic reason to be customized. Less is more - a clean default look is better than over-styling.

IMPORTANT: Always include ALL fields from the JSON schema below. Do not omit nested objects or fields, even when using defaults.

RESPOND WITH ONLY VALID JSON. No explanation, no markdown, just the JSON object.

Available options (use defaults unless the game strongly suggests otherwise):

corners.style: "none" | "brackets" | "flourish" | "arrows" | "dots"
  - none: No corner decorations (DEFAULT - use for most games)
  - brackets: L-shaped corners for tech/sci-fi
  - flourish: Ornate decorative corners for fantasy/medieval
  - arrows: Directional arrows for adventure/exploration
  - dots: Minimal dots for mystery/subtle themes

corners.color: "amber" | "emerald" | "cyan" | "violet" | "rose" | "slate"
  - amber: Warm gold/brown (DEFAULT)
  - emerald: Green for nature, adventure
  - cyan: Blue-green for sci-fi, technology
  - violet: Purple for magic, mystery
  - rose: Pink/red for romance, horror
  - slate: Gray for noir, minimal

background.animation: "none" | "stars" | "rain" | "fog" | "particles" | "scanlines"
  - none: No animation (DEFAULT - use for most games)
  - stars: Only for space/cosmic themes
  - rain: Only for noir/mystery/sad themes
  - fog: Only for horror/mysterious themes
  - particles: Only for fantasy/magical themes
  - scanlines: Only for retro/sci-fi themes

background.tint: "neutral" | "warm" | "cool" | "dark"
  - neutral: No tint (DEFAULT)
  - warm: Amber tint for cozy/fantasy
  - cool: Blue tint for tech/cold settings
  - dark: Darker for horror/noir

player.color: Same as corners.color (DEFAULT: "cyan")
player.indicator: "none" | "dot" | "arrow" | "chevron" | "diamond"
  - none: No indicator (DEFAULT)
  - dot/arrow/chevron/diamond: Only if it fits the theme
player.showChevron: true/false - show ">" prefix (DEFAULT: false)
  Only enable for sci-fi / hacker / terminal-style themes where the prompt-like ">" fits.
player.bgColor: "cyan" | "amber" | "violet" | "slate" | "white" | "emerald" | "rose" (DEFAULT: "cyan")

gameMessage.dropCap: true/false - decorative first letter (DEFAULT: true).
   Drop caps fit many story-like games, but disable them for modern/minimal UI vibes.
   Set dropCap = false for sci-fi / hacker / terminal themes, very short AI messages, or when typography.messages = "mono" / "sans" and you want a clean modern look.
gameMessage.dropCapColor: Same as corners.color. Only used if dropCap is true.
   Example (epic/classical): {"gameMessage": {"dropCap": true, "dropCapColor": "amber"}}

thinking.text: A thematic phrase shown while AI generates response (DEFAULT: "The story unfolds...")
thinking.style: "dots" | "spinner" | "pulse" | "typewriter" (DEFAULT: "dots")

typography.messages: "sans" | "serif" | "mono" | "fantasy"
  - sans: Modern clean feel (DEFAULT - use for most games)
  - serif: Classic book feel for fantasy/historical
  - mono: Terminal feel for tech/sci-fi
  - fantasy: Decorative for high fantasy only

statusEmojis: Object mapping status field names to emoji
  ONLY add emojis if they fit the status field. It's fine to leave fields without emoji.
  Leave empty {} if no obvious emoji mappings exist.
  Examples:
  - Health/HP/Life -> ❤️
  - Gold/Coins/Money -> 🪙
  - Energy/Stamina/Power -> ⚡
  - Mana/Magic -> 🔮
  - Food -> 🍖
  - Time -> ⏰

JSON SCHEMA:
{
  "corners": { "style": "string", "color": "string" },
  "background": { "animation": "string", "tint": "string" },
  "player": { "color": "string", "indicator": "string", "showChevron": boolean, "bgColor": "string" },
  "gameMessage": { "dropCap": boolean, "dropCapColor": "string" },
  "thinking": { "text": "string", "style": "string" },
  "typography": { "messages": "string" },
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
