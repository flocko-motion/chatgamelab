package engine

import (
	"os"
	"regexp"
	"slices"
	"testing"
)

// The editor offers what Themes lists and the playback-engine applies its own
// presets, so the two lists must be the same.
func TestThemesAreThePlaybackEnginesPresets(t *testing.T) {
	entry := regexp.MustCompile(`(?m)^  (\w+): \w+Preset,$`)
	var presets []string
	for _, file := range []string{"web/src/ui/theme/presets/simple.ts", "web/src/ui/theme/presets/animated.ts"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range entry.FindAllStringSubmatch(string(source), -1) {
			presets = append(presets, match[1])
		}
	}
	if !slices.Equal(presets, themeIDs) {
		t.Errorf("presets %v\n   themes %v", presets, themeIDs)
	}
}

func TestUnknownThemeIsAWarning(t *testing.T) {
	spec := adventure(MockOffer())
	spec.spec.Theme = "plaid"
	issues := check(spec)
	if hasError(issues) || !hasWarningAt(issues, "theme") {
		t.Errorf("issues = %v, want a warning on the theme", issues)
	}
	spec.spec.Theme = "noir"
	if hasWarningAt(check(spec), "theme") {
		t.Error("a known theme was warned about")
	}
}
