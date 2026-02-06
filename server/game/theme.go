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
1. Choose a preset that fits the game's theme, or use "custom" for unique themes.
2. If using a preset, you only need to specify "preset" and optionally override specific fields.
3. Include ALL fields when using preset: "custom". Do not omit any.
4. Output ONLY valid JSON. No explanation, no markdown.
5. ENSURE READABILITY: Never pair similar dark colors (e.g., dark bgColor with dark fontColor). Use high contrast:
   - Dark backgrounds (dark, black, blue, cyan, violet, rose) → use "light" fontColor
   - Light backgrounds (white, creme, *Light variants) → use "dark" fontColor
   - "hacker" fontColor only with black/dark backgrounds

AVAILABLE PRESETS:
- "default" - Neutral, warm amber accents
- "minimal" - Clean, slate gray, no decorations
- "medieval" - Medieval, flourish, serif font, drop caps
- "scifi" - Cyberpunk, brackets, cyan, mono font
- "horror" - Dark, no decorations, rose accents
- "adventure" - Exploration, arrows, emerald
- "mystery" - Mystic/supernatural, violet, ethereal
- "detective" - Noir/investigation, dark, amber accents
- "space" - Cosmic, dark background, cyan, stars
- "terminal" - Classic green on black, mono font
- "hacker" - Aggressive red AI / green user, mono font
- "playful" - Kids, colorful, star dividers
- "barbie" - Pink dream, rose, flourish, diamond dividers
- "nature" - Forest, emerald, flourish
- "ocean" - Underwater, cyan, cool tint
- "retro" - 80s, violet/cyan on dark
- "western" - Wild West, amber, warm
- "fire" -  dark, red accents, embers animation
- "desert" - Arid, sandy, hot climate, warm tones
- "tech" - Modern technology, clean digital, circuits animation
- "greenFantasy" - Enchanted forest, nature magic, sparkles animation
- "abstract" - Artistic, geometric shapes, geometric animation
- "romance" - Soft, warm, romantic, hearts animation
- "glitch" - Corrupted, digital chaos, glitch animation
- "snowy" - Winter wonderland, snow animation

OPTIONS (only needed for preset: "custom"):

corners.style: "none" (default) | "brackets" (tech) | "flourish" (fantasy) | "arrows" (exploration) | "dots" (mystery) | "dot" | "cursor"
corners.color: "amber" (default) | "emerald" | "cyan" | "violet" | "rose" | "slate" | "hacker"

background.tint: "neutral" (default) | "warm" | "cool" | "dark" | "black"
background.animation: "none" (default, preferred) | "stars" (space/scifi) | "bubbles" (ocean) | "fireflies" (fantasy/mystery) | "snow" | "matrix" (terminal/hacker) | "embers" (fire/destruction) | "hyperspace" (scifi/action) | "sparkles" (magic/fantasy) | "hearts" (romance) | "glitch" (corrupted/cyberpunk) | "circuits" (tech/digital) | "leaves" (nature/forest) | "geometric" (abstract/artistic) | "confetti" (playful/kids)
  NOTE: Only use animations if they fit the theme. Most games work good without animation. Prefer "none" unless the theme clearly benefits (e.g., space games with stars, hacker themes with matrix, fire/destruction with embers, cold scenarios with snow, water scenarios with bubbles, romance with hearts, tech with circuits).

player.color: Same colors as corners. Default: "cyan"
player.indicator: "none" (default) | "dot" | "arrow" | "chevron" | "diamond" | "cursor" | "underscore" | "pipe"
player.bgColor: "creme" (default) | "white" | "dark" | "black" | "blue"
player.fontColor: "dark" (default) | "light" | "hacker"
player.borderColor: Same colors as corners. Default: "cyan"

gameMessage.dropCap: true (default). Set false for terminal/mono/minimal styles.
gameMessage.dropCapColor: Same colors as corners. Default: matches corners.color.
gameMessage.bgColor: "white" (default) | "creme" | "dark" | "black" | "blue"
gameMessage.fontColor: "dark" (default) | "light" | "hacker"
gameMessage.borderColor: Same colors as corners. Default: "amber"

cards.borderThickness: "thin" (default) | "none" | "medium" | "thick"

thinking.text: Thematic loading phrase IN THE SAME LANGUAGE as the scenario. Default: "The story unfolds..."
thinking.style: "dots" (default) | "spinner" | "pulse" | "typewriter"
thinking.streamingCursor: "dots" (default) | "block" (█) | "pipe" (| great for terminal/hacker) | "underscore" (_) | "none"

typography.messages: "sans" (default) | "serif" (classic/fantasy) | "mono" (terminal) | "fantasy" (high fantasy only)

statusFields.bgColor: "creme" (default) | "white" | "dark" | "black" | "blue"
statusFields.accentColor: Same colors as corners. Default: "amber"
statusFields.borderColor: Same colors as corners. Default: "amber"
statusFields.fontColor: "dark" (default) | "light" | "hacker"

header.bgColor: "white" (default) | "creme" | "dark" | "black"
header.fontColor: "dark" (default) | "light" | "hacker"
header.accentColor: Same colors as corners. Default: "amber"

divider.style: "dot" (default) | "dots" | "line" | "diamond" | "star" | "dash" | "none"
divider.color: Same colors as corners. Default: "amber"

statusEmojis: Map status field names to emoji. Use {} if no obvious mappings.
  Common: Health→❤️, Gold→🪙, Energy→⚡, Mana→🔮, Food→🍖, Time→⏰

STRUCTURE:
{
  "preset": "<preset_name>",
  "override": { ...only fields you want to change from preset defaults... }
}

EXAMPLE 1 (preset only - PREFERRED, simplest):
{
  "preset": "space"
}

EXAMPLE 2 (preset with overrides):
{
  "preset": "fantasy",
  "override": {
    "thinking": { "text": "The tale continues..." },
    "statusEmojis": { "Health": "❤️", "Gold": "🪙" }
  }
}

EXAMPLE 3 (custom theme - all fields required in override):
{
  "preset": "custom",
  "override": {
    "corners": { "style": "none", "color": "amber" },
    "background": { "tint": "neutral" },
    "player": { "color": "cyan", "indicator": "none", "bgColor": "creme", "fontColor": "dark", "borderColor": "cyan" },
    "gameMessage": { "dropCap": true, "dropCapColor": "amber", "bgColor": "white", "fontColor": "dark", "borderColor": "amber" },
    "cards": { "borderThickness": "thin" },
    "thinking": { "text": "The story unfolds...", "style": "dots", "streamingCursor": "dots" },
    "typography": { "messages": "sans" },
    "statusFields": { "bgColor": "creme", "accentColor": "amber", "borderColor": "amber", "fontColor": "dark" },
    "header": { "bgColor": "white", "fontColor": "dark", "accentColor": "amber" },
    "divider": { "style": "dot", "color": "amber" },
    "statusEmojis": {}
  }
}`

// GenerateTheme generates a visual theme for the game based on its description
func GenerateTheme(ctx context.Context, session *obj.GameSession, game *obj.Game) (*obj.GameTheme, error) {
	if session == nil || session.ApiKey == nil {
		return nil, fmt.Errorf("session or API key is nil")
	}

	log.Debug("generating theme for game", "game_id", game.ID, "game_name", game.Name)

	// Get AI platform
	platform, err := ai.GetAiPlatform(session.AiPlatform)
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

// validPresets lists all valid preset names
var validPresets = map[string]bool{
	"default": true, "minimal": true, "fantasy": true, "scifi": true, "horror": true,
	"adventure": true, "mystery": true, "detective": true, "space": true,
	"terminal": true, "hacker": true,
	"playful": true, "barbie": true, "nature": true, "ocean": true, "retro": true,
	"western": true, "fire": true, "desert": true, "tech": true, "greenFantasy": true,
	"abstract": true, "romance": true, "glitch": true, "snowy": true, "custom": true,
}

// validateTheme ensures all theme values are valid
func validateTheme(theme obj.GameTheme) obj.GameTheme {
	// Validate preset name (default to "default" if invalid)
	if !validPresets[theme.Preset] {
		theme.Preset = "default"
	}

	// Validate override fields if present
	if theme.Override != nil {
		theme.Override = validateOverride(theme.Override)
	}

	return theme
}

// validateOverride validates the override fields
func validateOverride(override *obj.GameThemeOverride) *obj.GameThemeOverride {
	validColors := map[string]bool{
		"amber": true, "emerald": true, "cyan": true, "violet": true, "rose": true, "slate": true,
		"hacker": true, "terminal": true,
		"brown": true, "brownLight": true, "pink": true, "pinkLight": true, "orange": true, "orangeLight": true,
	}
	validBgColors := map[string]bool{
		"white": true, "creme": true, "dark": true, "black": true,
		"blue": true, "blueLight": true, "green": true, "greenLight": true, "red": true, "redLight": true,
		"amber": true, "amberLight": true, "violet": true, "violetLight": true, "rose": true, "roseLight": true, "cyan": true, "cyanLight": true,
		"pink": true, "pinkLight": true, "orange": true, "orangeLight": true,
	}
	validFontColors := map[string]bool{
		"dark": true, "light": true, "hacker": true, "terminal": true,
		"pink": true, "amber": true, "cyan": true, "violet": true,
	}
	validStyles := map[string]bool{"none": true, "brackets": true, "flourish": true, "arrows": true, "dots": true, "dot": true, "cursor": true}
	validTints := map[string]bool{
		"neutral": true, "warm": true, "cool": true, "dark": true, "black": true,
		"pink": true, "green": true, "blue": true, "violet": true,
		"darkCyan": true, "darkViolet": true, "darkBlue": true, "darkRose": true,
	}
	validIndicators := map[string]bool{"none": true, "dot": true, "arrow": true, "chevron": true, "diamond": true, "cursor": true, "underscore": true, "pipe": true, "star": true}
	validThicknesses := map[string]bool{"none": true, "thin": true, "medium": true, "thick": true}
	validThinkingStyles := map[string]bool{"dots": true, "spinner": true, "pulse": true, "typewriter": true}
	validStreamingCursors := map[string]bool{"dots": true, "block": true, "pipe": true, "underscore": true, "none": true}
	validFonts := map[string]bool{"serif": true, "sans": true, "mono": true, "fantasy": true}
	validDividerStyles := map[string]bool{"dot": true, "dots": true, "line": true, "diamond": true, "star": true, "dash": true, "none": true}

	// Validate corners
	if override.Corners != nil {
		if override.Corners.Style != "" && !validStyles[override.Corners.Style] {
			override.Corners.Style = ""
		}
		if override.Corners.Color != "" && !validColors[override.Corners.Color] {
			override.Corners.Color = ""
		}
	}

	// Validate background
	validAnimations := map[string]bool{"none": true, "stars": true, "bubbles": true, "fireflies": true, "snow": true, "matrix": true, "embers": true, "hyperspace": true, "sparkles": true, "hearts": true, "glitch": true, "circuits": true, "leaves": true, "geometric": true, "confetti": true}
	if override.Background != nil {
		if override.Background.Tint != "" && !validTints[override.Background.Tint] {
			override.Background.Tint = ""
		}
		if override.Background.Animation != "" && !validAnimations[override.Background.Animation] {
			override.Background.Animation = ""
		}
	}

	// Validate player
	if override.Player != nil {
		if override.Player.Color != "" && !validColors[override.Player.Color] {
			override.Player.Color = ""
		}
		if override.Player.Indicator != "" && !validIndicators[override.Player.Indicator] {
			override.Player.Indicator = ""
		}
		if override.Player.BgColor != "" && !validBgColors[override.Player.BgColor] {
			override.Player.BgColor = ""
		}
		if override.Player.FontColor != "" && !validFontColors[override.Player.FontColor] {
			override.Player.FontColor = ""
		}
		if override.Player.BorderColor != "" && !validColors[override.Player.BorderColor] {
			override.Player.BorderColor = ""
		}
	}

	// Validate game message
	if override.GameMessage != nil {
		if override.GameMessage.DropCapColor != "" && !validColors[override.GameMessage.DropCapColor] {
			override.GameMessage.DropCapColor = ""
		}
		if override.GameMessage.BgColor != "" && !validBgColors[override.GameMessage.BgColor] {
			override.GameMessage.BgColor = ""
		}
		if override.GameMessage.FontColor != "" && !validFontColors[override.GameMessage.FontColor] {
			override.GameMessage.FontColor = ""
		}
		if override.GameMessage.BorderColor != "" && !validColors[override.GameMessage.BorderColor] {
			override.GameMessage.BorderColor = ""
		}
	}

	// Validate cards
	if override.Cards != nil {
		if override.Cards.BorderThickness != "" && !validThicknesses[override.Cards.BorderThickness] {
			override.Cards.BorderThickness = ""
		}
	}

	// Validate thinking
	if override.Thinking != nil {
		if override.Thinking.Style != "" && !validThinkingStyles[override.Thinking.Style] {
			override.Thinking.Style = ""
		}
		if override.Thinking.StreamingCursor != "" && !validStreamingCursors[override.Thinking.StreamingCursor] {
			override.Thinking.StreamingCursor = ""
		}
	}

	// Validate typography
	if override.Typography != nil {
		if override.Typography.Messages != "" && !validFonts[override.Typography.Messages] {
			override.Typography.Messages = ""
		}
	}

	// Validate status fields
	if override.StatusFields != nil {
		if override.StatusFields.BgColor != "" && !validBgColors[override.StatusFields.BgColor] {
			override.StatusFields.BgColor = ""
		}
		if override.StatusFields.AccentColor != "" && !validColors[override.StatusFields.AccentColor] {
			override.StatusFields.AccentColor = ""
		}
		if override.StatusFields.BorderColor != "" && !validColors[override.StatusFields.BorderColor] {
			override.StatusFields.BorderColor = ""
		}
		if override.StatusFields.FontColor != "" && !validFontColors[override.StatusFields.FontColor] {
			override.StatusFields.FontColor = ""
		}
	}

	// Validate header
	if override.Header != nil {
		if override.Header.BgColor != "" && !validBgColors[override.Header.BgColor] {
			override.Header.BgColor = ""
		}
		if override.Header.FontColor != "" && !validFontColors[override.Header.FontColor] {
			override.Header.FontColor = ""
		}
		if override.Header.AccentColor != "" && !validColors[override.Header.AccentColor] {
			override.Header.AccentColor = ""
		}
	}

	// Validate divider
	if override.Divider != nil {
		if override.Divider.Style != "" && !validDividerStyles[override.Divider.Style] {
			override.Divider.Style = ""
		}
		if override.Divider.Color != "" && !validColors[override.Divider.Color] {
			override.Divider.Color = ""
		}
	}

	return override
}

// defaultTheme returns the default neutral theme
func defaultTheme() *obj.GameTheme {
	return &obj.GameTheme{
		Preset: "default",
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
