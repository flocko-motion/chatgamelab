// package: game / narration clean-up
// type:    logic
// job:     strips JSON the AI sometimes puts into the narration step (whole answer as JSON, or a status block appended to the prose) and markdown emphasis stars.
// limits:  pure text clean-up; does not call the AI or touch the stream (-> session_play.go).
package game

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Paired markdown emphasis on one line: **bold** and *italic*. The content must not start or end
// with a space, so a lone star or "5 * 3" stays. Bold runs first, so ***x*** ends up as x.
var (
	markdownBold   = regexp.MustCompile(`\*\*([^\s*](?:[^*\n]*[^\s*])?)\*\*`)
	markdownItalic = regexp.MustCompile(`\*([^\s*](?:[^*\n]*[^\s*])?)\*`)
)

// stripMarkdownEmphasis removes paired ** and * markers but keeps the text between them.
// The narration prompt forbids markdown, but some models (mostly Mistral) still use it, and the
// player shows plain text.
func stripMarkdownEmphasis(text string) string {
	text = markdownBold.ReplaceAllString(text, "$1")
	return markdownItalic.ReplaceAllString(text, "$1")
}

// narrationKeys are the fields in which the AI usually puts the prose when it wrongly answers
// the narration step with a JSON object. They are tried in this order.
var narrationKeys = []string{"narration", "scene", "story", "prose", "text", "message", "summary"}

// cleanNarration returns the narration without JSON. The narration step must be plain prose,
// but now and then the model (mostly at the opening scene) answers with a JSON object such as
// {"narration":"...","status":{...}}, or appends a status JSON block to the prose. Markdown
// emphasis stars are removed as well. Clean text comes back unchanged (apart from surrounding whitespace).
func cleanNarration(text string) string {
	return stripMarkdownEmphasis(stripJSON(text))
}

// stripJSON removes JSON the model put into the narration (see cleanNarration).
func stripJSON(text string) string {
	trimmed := strings.TrimSpace(text)
	// A trailing markdown fence (```json ... ```) around the JSON is dropped first.
	unfenced := strings.TrimSpace(strings.TrimSuffix(trimmed, "```"))

	// Case 1: the whole answer is a JSON object -> take the prose out of it.
	if whole := trimFenceOpener(unfenced); strings.HasPrefix(whole, "{") {
		var obj map[string]any
		if err := json.Unmarshal([]byte(whole), &obj); err == nil {
			if prose := proseFromObject(obj); prose != "" {
				return prose
			}
		}
	}

	// Case 2: prose followed by a JSON block -> cut the block off.
	// Scanning from the left finds the outermost block.
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

// trimFenceOpener removes a leading ``` or ```json line.
func trimFenceOpener(text string) string {
	if !strings.HasPrefix(text, "```") {
		return text
	}
	if nl := strings.IndexByte(text, '\n'); nl >= 0 {
		return strings.TrimSpace(text[nl+1:])
	}
	return text
}

// proseFromObject picks the prose from a JSON object: the first known narration field,
// otherwise the longest top-level string value.
func proseFromObject(obj map[string]any) string {
	for _, key := range narrationKeys {
		for k, v := range obj {
			if strings.EqualFold(k, key) {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	longest := ""
	for _, v := range obj {
		if s, ok := v.(string); ok && len(s) > len(longest) {
			longest = s
		}
	}
	return strings.TrimSpace(longest)
}
