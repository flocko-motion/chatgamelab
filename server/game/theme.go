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
Based on the game's name, description, and setting, generate a JSON theme configuration.

The theme controls the visual appearance of the game player:
- Corner decorations on scene cards
- Background animation effects
- Player message styling (the text input from the player)
- "AI thinking" indicator text
- Typography
- Emoji icons for status fields

RESPOND WITH ONLY VALID JSON. No explanation, no markdown, just the JSON object.

Available options:

corners.style: "brackets" | "flourish" | "arrows" | "dots" | "none"
  - brackets: Clean tech/sci-fi look with L-shaped corners
  - flourish: Ornate fantasy/medieval decorative corners
  - arrows: Directional arrows for adventure/exploration
  - dots: Minimal dots for mystery/subtle themes
  - none: No corner decorations

corners.color: "amber" | "emerald" | "cyan" | "violet" | "rose" | "slate"
  - amber: Warm gold/brown for fantasy, western, historical
  - emerald: Green for nature, adventure, exploration
  - cyan: Blue-green for sci-fi, technology, ocean
  - violet: Purple for magic, mystery, supernatural
  - rose: Pink/red for romance, horror, drama
  - slate: Gray for noir, mystery, minimal

background.animation: "none" | "stars" | "rain" | "fog" | "particles" | "scanlines"
  - none: Clean, no animation
  - stars: Twinkling stars for space/cosmic themes
  - rain: Falling rain for noir/mystery/sad themes
  - fog: Drifting fog for horror/mysterious themes
  - particles: Floating particles for fantasy/magical themes
  - scanlines: CRT scanlines for retro/sci-fi themes

background.tint: "warm" | "cool" | "neutral" | "dark"
  - warm: Amber tinted for cozy/fantasy
  - cool: Blue tinted for tech/cold settings
  - neutral: No tint
  - dark: Darker background for horror/noir

player.color: Same options as corners.color - this colors the player's input text
player.indicator: "dot" | "arrow" | "chevron" | "diamond" | "none" - icon before player text
player.monochrome: true/false - whether player text is single color or gradient
player.showChevron: true/false - show ">" prefix before player text

thinking.text: A short thematic phrase shown while AI generates response
  Examples: "The story unfolds...", "Processing...", "The tale continues...", "Investigating..."
thinking.style: "dots" | "spinner" | "pulse" | "typewriter"

typography.messages: "serif" | "sans" | "mono" | "fantasy"
  - serif: Classic book feel
  - sans: Modern clean feel
  - mono: Terminal/tech feel
  - fantasy: Decorative fantasy feel

statusEmojis: Object mapping status field names to emoji
  Example: {"Health": "❤️", "Gold": "🪙", "Energy": "⚡"}
  Use appropriate emoji for the game's status fields. Common mappings:
  - Health/HP/Life -> ❤️
  - Gold/Coins/Money -> 🪙
  - Energy/Stamina/Power -> ⚡
  - Mana/Magic -> 🔮
  - Time -> ⏰
  - Reputation/Fame -> ⭐
  - Fear/Terror -> 😨
  - Hunger/Food -> 🍖
  - Oxygen/Air -> 💨
  - Ammo/Bullets -> 🔫
  - Shield/Armor -> 🛡️
  - Speed -> 🏃
  - Luck -> 🍀

JSON SCHEMA:
{
  "corners": { "style": "string", "color": "string" },
  "background": { "animation": "string", "tint": "string" },
  "player": { "color": "string", "indicator": "string", "monochrome": boolean, "showChevron": boolean },
  "thinking": { "text": "string", "style": "string" },
  "typography": { "messages": "string" },
  "statusEmojis": { "fieldName": "emoji", ... }
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
	var statusFieldsList string
	if game.StatusFields != "" {
		statusFieldsList = game.StatusFields
	} else {
		statusFieldsList = "None defined"
	}

	userPrompt := fmt.Sprintf(`Generate a visual theme for this game:

Game Name: %s
Description: %s
Setting/Scenario: %s
Status Fields: %s

Generate the JSON theme that best matches this game's atmosphere and genre.`,
		game.Name,
		game.Description,
		truncateString(game.SystemMessageScenario, 500),
		statusFieldsList,
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
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	}
	if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}
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
	// Validate corner style
	validStyles := map[string]bool{"brackets": true, "flourish": true, "arrows": true, "dots": true, "none": true}
	if !validStyles[theme.Corners.Style] {
		theme.Corners.Style = "brackets"
	}

	// Validate colors
	validColors := map[string]bool{"amber": true, "emerald": true, "cyan": true, "violet": true, "rose": true, "slate": true}
	if !validColors[theme.Corners.Color] {
		theme.Corners.Color = "amber"
	}
	if !validColors[theme.Player.Color] {
		theme.Player.Color = "cyan"
	}

	// Validate background animation
	validAnimations := map[string]bool{"none": true, "stars": true, "rain": true, "fog": true, "particles": true, "scanlines": true}
	if !validAnimations[theme.Background.Animation] {
		theme.Background.Animation = "none"
	}

	// Validate background tint
	validTints := map[string]bool{"warm": true, "cool": true, "neutral": true, "dark": true}
	if !validTints[theme.Background.Tint] {
		theme.Background.Tint = "warm"
	}

	// Validate player indicator
	validIndicators := map[string]bool{"dot": true, "arrow": true, "chevron": true, "diamond": true, "none": true}
	if !validIndicators[theme.Player.Indicator] {
		theme.Player.Indicator = "dot"
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

// defaultTheme returns the default theme (current sci-fi/tech style)
func defaultTheme() *obj.GameTheme {
	return &obj.GameTheme{
		Corners: obj.GameThemeCorners{
			Style: "brackets",
			Color: "amber",
		},
		Background: obj.GameThemeBackground{
			Animation: "none",
			Tint:      "warm",
		},
		Player: obj.GameThemePlayer{
			Color:       "cyan",
			Indicator:   "dot",
			Monochrome:  true,
			ShowChevron: true,
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
