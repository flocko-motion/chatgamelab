package genre

import "strings"

// A game's cover is one picture, made outside any session: the image style
// adapted to the youth-protection rule, a long scenario condensed, and the
// picture drawn inside the cover frame. Preparation's prompts are Adventure's,
// so a cover looks like the game's own pictures.

// coverFrame is the fixed frame of a cover's picture prompt: the game's
// details go between its first line and its last.
const coverFrame = "You are generating the cover art for an interactive story game: one picture that shows what the game is about and makes people want to play it.\n" +
	"Important: no text, no letters, no title in the picture."

// CoverStylePrompt adapts the image style to the youth-protection rule; its
// message is CoverStyleMessage.
func CoverStylePrompt() string { return AdventurePrompts[PromptImageStyle] }

// CoverCondensePrompt condenses a long scenario; the scenario is its message.
func CoverCondensePrompt() string { return AdventurePrompts[PromptCondenseScenario] }

// CoverStyleMessage is the style adaptation's message: the style, or
// AdventureImageStyle when the game names none, and the rule.
func CoverStyleMessage(style, guardrail string) string { return styleMessage(style, guardrail) }

// CoverNeedsCondensing says whether a scenario is long enough to be condensed
// before it goes into the picture prompt.
func CoverNeedsCondensing(scenario string) bool { return len(scenario) > longScenario }

// CoverPrompt builds a cover's picture prompt, in the order the image model
// reads best, as a scene's: the frame's first line, the title and the idea,
// the scenario, the designer's hint, the style, and the frame's last line.
// An empty style takes AdventureImageStyle.
func CoverPrompt(title, description, scenario, hint, style string) string {
	lines := strings.Split(coverFrame, "\n")
	parts := []string{lines[0]}
	if t := strings.TrimSpace(title); t != "" {
		parts = append(parts, "The game is called: "+t)
	}
	if d := strings.TrimSpace(description); d != "" {
		parts = append(parts, "The game idea is: "+d)
	}
	if s := strings.TrimSpace(scenario); s != "" {
		parts = append(parts, "The game scenario is: "+s)
	}
	if h := strings.TrimSpace(hint); h != "" {
		parts = append(parts, "The cover should show: "+h)
	}
	if strings.TrimSpace(style) == "" {
		style = AdventureImageStyle
	}
	parts = append(parts, "The artistic style should be: "+style)
	parts = append(parts, lines[len(lines)-1])
	return strings.Join(parts, "\n")
}
