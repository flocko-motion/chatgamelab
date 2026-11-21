package genre

import (
	"encoding/json"
	"regexp"
	"strings"
)

// cleanNarration is Expand's prose without what the narration prompt forbids
// but a model now and then writes, mostly at the opening: the whole answer as
// a JSON object such as {"narration": "…", "status": {…}}, a JSON block after
// the prose, and markdown emphasis. Clean prose comes back as it is, apart
// from the space around it.
func cleanNarration(text string) string {
	return stripMarkdownEmphasis(stripJSON(text))
}

// Paired markdown emphasis on one line: **bold** and *italic*. The content may
// not start or end with a space, so a lone star or "5 * 3" stays. Bold goes
// first, so ***x*** ends up as x.
var (
	markdownBold   = regexp.MustCompile(`\*\*([^\s*](?:[^*\n]*[^\s*])?)\*\*`)
	markdownItalic = regexp.MustCompile(`\*([^\s*](?:[^*\n]*[^\s*])?)\*`)
)

// stripMarkdownEmphasis drops paired ** and * and keeps the text between them:
// the player shows plain text.
func stripMarkdownEmphasis(text string) string {
	text = markdownBold.ReplaceAllString(text, "$1")
	return markdownItalic.ReplaceAllString(text, "$1")
}

// narrationKeys are the fields a model puts the prose in when it answers the
// narration with a JSON object, tried in this order.
var narrationKeys = []string{"narration", "scene", "story", "prose", "text", "message", "summary"}

// stripJSON takes the prose out of a JSON answer, or cuts a JSON block off
// the end of the prose.
func stripJSON(text string) string {
	trimmed := strings.TrimSpace(text)
	// A closing markdown fence after the JSON goes first.
	unfenced := strings.TrimSpace(strings.TrimSuffix(trimmed, "```"))

	// The whole answer is a JSON object: the prose is in one of its fields.
	if whole := trimFenceOpener(unfenced); strings.HasPrefix(whole, "{") {
		var object map[string]any
		if err := json.Unmarshal([]byte(whole), &object); err == nil {
			if prose := proseFromObject(object); prose != "" {
				return prose
			}
		}
	}

	// Prose followed by a JSON block: the block goes. Scanning from the left
	// finds the outermost one.
	if strings.HasSuffix(unfenced, "}") {
		for i := 0; i < len(unfenced); i++ {
			if unfenced[i] != '{' {
				continue
			}
			before := strings.TrimSpace(unfenced[:i])
			before = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(before, "```json"), "```"))
			if before != "" && json.Valid([]byte(unfenced[i:])) {
				return before
			}
		}
	}

	return trimmed
}

// trimFenceOpener drops a leading ``` or ```json line.
func trimFenceOpener(text string) string {
	if !strings.HasPrefix(text, "```") {
		return text
	}
	if nl := strings.IndexByte(text, '\n'); nl >= 0 {
		return strings.TrimSpace(text[nl+1:])
	}
	return text
}

// proseFromObject picks the prose from a JSON object: the first known
// narration field, otherwise the longest string at its top level.
func proseFromObject(object map[string]any) string {
	for _, key := range narrationKeys {
		for k, v := range object {
			if strings.EqualFold(k, key) {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	longest := ""
	for _, v := range object {
		if s, ok := v.(string); ok && len(s) > len(longest) {
			longest = s
		}
	}
	return strings.TrimSpace(longest)
}
