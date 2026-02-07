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
const ThemeGenerationPrompt = `You are a visual theme generator for a text adventure game. Pick the best preset and generate a minimal JSON theme.

RULES:
1. Choose the preset that best fits the game's genre, setting, and mood.
2. Output ONLY valid JSON. No explanation, no markdown.
3. The "thinkingText" MUST be in the SAME LANGUAGE as the game scenario.

AVAILABLE PRESETS:
- "default" - Neutral, clean, slate accents, no animation
- "minimal" - Clean, slate gray, no decorations, no animation
- "scifi" - Clean futuristic (Star Trek), blue/cyan, mono font, stars animation
- "cyberpunk" - Neon-soaked gritty streets, pink/cyan on black, glitch animation
- "medieval" - Medieval fantasy, flourish corners, serif font, drop caps, fireflies
- "horror" - Dark, minimal, rose accents, no animation
- "adventure" - Exploration, arrows, emerald, nature feel, no animation
- "mystery" - Whodunit/suspense, dark blue atmosphere, fireflies
- "mystic" - Occult/arcane, ethereal purple magic, sparkles
- "detective" - Classic whodunit, warm amber tones, serif, no animation
- "noir" - Dark moody, black & white stark contrast, no animation
- "space" - Cosmic, dark background, cyan, hyperspace animation
- "terminal" - Classic green on black, mono font, matrix rain
- "hacker" - Aggressive red AI / green user, mono font, matrix rain
- "playful" - Kids, colorful, confetti animation
- "barbie" - Pink dream, flourish, diamond dividers, no animation
- "nature" - Forest, emerald, flourish, falling leaves
- "ocean" - Surface/coastal, bright cyan, cool tint, bubbles
- "underwater" - Deep sea, dark, bioluminescent cyan, bubbles
- "pirate" - Nautical adventure, amber on dark blue, stars
- "retro" - 80s, violet/cyan on dark, no animation
- "western" - Wild West, amber, warm tones, no animation
- "fire" - Dark, orange/red accents, embers animation
- "desert" - Arid, sandy, warm tones, no animation
- "tech" - Modern technology, clean digital, circuits animation
- "greenFantasy" - Enchanted forest, nature magic, sparkles
- "abstract" - Artistic, geometric shapes, geometric animation
- "romance" - Soft, warm, romantic, hearts animation
- "glitch" - Corrupted, digital chaos, glitch animation
- "snowy" - Winter wonderland, snow animation
- "fairy" - Light magical, pastel pink/violet, sparkles
- "steampunk" - Brass, gears, Victorian industrial, no animation
- "zombie" - Post-apocalyptic, decayed, eerie green, no animation
- "school" - Friendly, educational, clean sky blue/white, no animation
- "candy" - Sweet, colorful pastels, coral/lavender, no animation
- "superhero" - Bold, comic book, indigo/coral, no animation
- "sunshine" - Warm, bright, cheerful yellow, no animation
- "storybook" - Classic children's book, teal/coral, warm, no animation
- "jungle" - Tropical, lush, vibrant lime/green, falling leaves
- "garden" - Blooming flowers, soft teal/coral, sparkles
- "circus" - Bright, bold, showtime energy, confetti

AVAILABLE ANIMATIONS (for optional override):
"none", "stars", "bubbles", "fireflies", "snow", "matrix", "embers", "hyperspace", "sparkles", "hearts", "glitch", "circuits", "leaves", "geometric", "confetti"

OUTPUT FORMAT:
{
  "preset": "<preset_name>",
  "thinkingText": "<thematic loading phrase in scenario language>",
  "statusEmojis": { "<FieldName>": "<emoji>", ... },
  "animation": "<optional: only if you want to override the preset's default animation>"
}

EXAMPLES:

Example 1 - Space game:
{ "preset": "space", "thinkingText": "Scanning the cosmos...", "statusEmojis": { "Fuel": "⛽", "Hull": "🛡️" } }

Example 2 - German medieval game:
{ "preset": "medieval", "thinkingText": "Die Geschichte entfaltet sich...", "statusEmojis": { "Gesundheit": "❤️", "Gold": "🪙" } }

Example 3 - Horror with snow override:
{ "preset": "horror", "thinkingText": "Something stirs...", "statusEmojis": { "Sanity": "🧠" }, "animation": "snow" }

Example 4 - Simple kids game (no status fields):
{ "preset": "playful", "thinkingText": "Magic is happening..." }

Example 5 - Cyberpunk game:
{ "preset": "cyberpunk", "thinkingText": "Jacking in...", "statusEmojis": { "Credits": "💰", "Rep": "📡" } }

Example 6 - Pirate game in German:
{ "preset": "pirate", "thinkingText": "Kurs wird berechnet...", "statusEmojis": { "Gold": "🪙", "Rum": "🍺" } }

COMMON EMOJI MAPPINGS:
Health/Leben→❤️, Gold/Münzen→🪙, Energy/Energie→⚡, Mana→🔮, Food/Essen→🍖, Time/Zeit→⏰, Armor/Rüstung→🛡️, Strength/Stärke→💪, Luck/Glück→🍀, Score/Punkte→⭐`

// GenerateTheme generates a visual theme for the game based on its description
func GenerateTheme(ctx context.Context, session *obj.GameSession, game *obj.Game) (*obj.GameTheme, obj.TokenUsage, error) {
	if session == nil || session.ApiKey == nil {
		return nil, obj.TokenUsage{}, fmt.Errorf("session or API key is nil")
	}

	log.Debug("generating theme for game", "game_id", game.ID, "game_name", game.Name)

	// Get AI platform
	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		log.Debug("failed to get AI platform for theme generation", "error", err)
		return nil, obj.TokenUsage{}, fmt.Errorf("failed to get AI platform: %w", err)
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
	response, usage, err := platform.GenerateTheme(ctx, session, ThemeGenerationPrompt, userPrompt)
	if err != nil {
		log.Debug("AI theme generation failed", "error", err)
		return nil, usage, fmt.Errorf("failed to generate theme: %w", err)
	}

	// Parse the JSON response
	theme, err := parseThemeResponse(response)
	if err != nil {
		log.Debug("failed to parse theme response", "error", err, "response", response)
		// Return default theme on parse error
		return defaultTheme(), usage, nil
	}

	log.Debug("theme generated successfully", "theme", theme)
	return theme, usage, nil
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
	"default": true, "minimal": true, "scifi": true, "cyberpunk": true, "horror": true,
	"adventure": true, "mystery": true, "mystic": true, "detective": true, "noir": true,
	"space": true, "terminal": true, "hacker": true, "medieval": true,
	"playful": true, "barbie": true, "nature": true, "ocean": true, "underwater": true,
	"pirate": true, "retro": true, "western": true, "fire": true, "desert": true,
	"tech": true, "greenFantasy": true, "abstract": true, "romance": true,
	"glitch": true, "snowy": true, "fairy": true, "steampunk": true, "zombie": true,
	"school": true, "candy": true, "superhero": true, "sunshine": true, "storybook": true,
	"jungle": true, "garden": true, "circus": true,
}

// validAnimations lists all valid animation names
var validAnimations = map[string]bool{
	"none": true, "stars": true, "bubbles": true, "fireflies": true, "snow": true,
	"matrix": true, "embers": true, "hyperspace": true, "sparkles": true, "hearts": true,
	"glitch": true, "circuits": true, "leaves": true, "geometric": true, "confetti": true,
}

// validateTheme ensures all theme values are valid
func validateTheme(theme obj.GameTheme) obj.GameTheme {
	// Validate preset name (default to "default" if invalid)
	if !validPresets[theme.Preset] {
		theme.Preset = "default"
	}

	// Validate animation override (clear if invalid)
	if theme.Animation != "" && !validAnimations[theme.Animation] {
		theme.Animation = ""
	}

	return theme
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
