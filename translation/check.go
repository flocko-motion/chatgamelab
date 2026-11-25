package main

import (
	"fmt"
	"regexp"
	"sort"
)

// placeholderRE finds the placeholders a text fills in, such as {{siteName}}.
var placeholderRE = regexp.MustCompile(`\{\{\s*[^{}]+?\s*\}\}`)

// controlRE finds control characters, which no UI text holds but a line
// break or a tab: they are what is left of markup a translation broke.
var controlRE = regexp.MustCompile("[\x00-\x08\x0b-\x1f\x7f]")

// tagRE finds the markup a text carries for the web app, such as <mark>.
var tagRE = regexp.MustCompile(`</?[A-Za-z][\w-]*>`)

// placeholders are the placeholders and tags in s, sorted, each as often as
// it occurs: a translation keeps them all.
func placeholders(s string) []string {
	found := append(placeholderRE.FindAllString(s, -1), tagRE.FindAllString(s, -1)...)
	sort.Strings(found)
	return found
}

// checkTexts names what is wrong with texts, translations of the fields of
// want (path → English text): a field missing, one not asked for, or one
// whose placeholders or tags differ from the English, or that holds control
// characters. An empty text may be right: a
// sentence's fragment some languages leave out.
func checkTexts(want, texts map[string]string) []string {
	var problems []string
	for path, en := range want {
		got, ok := texts[path]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: missing", path))
		case controlRE.MatchString(got):
			problems = append(problems, fmt.Sprintf("%s: control characters %q", path, got))
		case fmt.Sprint(placeholders(got)) != fmt.Sprint(placeholders(en)):
			problems = append(problems, fmt.Sprintf("%s: placeholders %v, the English has %v", path, placeholders(got), placeholders(en)))
		}
	}
	for path := range texts {
		if _, ok := want[path]; !ok {
			problems = append(problems, fmt.Sprintf("%s: not asked for", path))
		}
	}
	sort.Strings(problems)
	return problems
}

// fillPaths is tree (a locale file, or part of one) with the leaves named in
// values set to them; other leaves stay.
func fillPaths(tree string, values map[string]string) (string, error) {
	v, err := parse(tree)
	if err != nil {
		return "", err
	}
	var walk func(v any, path string) any
	walk = func(v any, path string) any {
		switch val := v.(type) {
		case map[string]any:
			for k, child := range val {
				val[k] = walk(child, join(path, k))
			}
			return val
		case []any:
			for i, child := range val {
				val[i] = walk(child, fmt.Sprintf("%s[%d]", path, i))
			}
			return val
		default:
			if s, ok := values[path]; ok {
				return s
			}
			return val
		}
	}
	return mustIndent(walk(v, "")), nil
}
