package genre

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

// Prompts are the prompts a genre's mechanics need, keyed by name.
//
// Two authors write the text a session runs on, and they are not the same
// person. A game designer writes what a particular game is about — the
// scenario — and those arrive in the spec. The mechanics designer writes how a
// genre works at all, and those live here, in the genre's own Go file, because
// a genre is code.
//
// They are a named map rather than string constants for three reasons: a view
// can show what a session is actually running on, which for a platform that
// teaches how AI works is the subject rather than a debug aid; a spec can
// replace one by name without a new build; and a prompt nobody can see is a
// prompt nobody can review.
type Prompts map[string]string

// Resolve applies a spec's overrides to a genre's defaults.
//
// An unknown name is an error rather than a no-op. Overriding by name is
// untyped by construction, so a misspelling would otherwise be silent — the
// session would run on the default while its author believed it was running on
// something else, which is the worst of both.
func Resolve(defaults Prompts, overrides map[string]string) (Prompts, error) {
	resolved := maps.Clone(defaults)
	if resolved == nil {
		resolved = Prompts{}
	}

	var unknown []string
	for name, text := range overrides {
		if _, known := resolved[name]; !known {
			unknown = append(unknown, name)
			continue
		}
		resolved[name] = text
	}

	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("no such prompt: %s (this genre has %s)",
			strings.Join(unknown, ", "), strings.Join(sortedNames(defaults), ", "))
	}
	return resolved, nil
}

func sortedNames(p Prompts) []string {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
