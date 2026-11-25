package main

// Proposals are improvements Claude suggests to existing translations. They
// are never applied unasked: they go into a checklist, proposalsFile, where a
// person ticks the ones to take, and --apply applies the ticked ones.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// proposalsFile is the checklist, kept out of git.
const proposalsFile = "translation/proposals.md"

// proposal is one suggested improvement of a language's text.
type proposal struct {
	Lang     string
	Path     string
	Now      string // the text when it was proposed; a changed one is stale
	Proposed string
	Reason   string
	Ticked   bool
}

func (p proposal) key() string { return p.Lang + "\x00" + p.Path + "\x00" + p.Proposed }

const proposalsHeader = `# Translation proposals

Claude suggests these improvements to existing translations. Tick the ones to
take ("- [x]"), then run ` + "`make translate-apply`" + `. Unticked ones are dropped.
`

// writeProposals writes ps to the checklist, by language and path.
func writeProposals(ps []proposal) {
	ps = append([]proposal(nil), ps...)
	sort.Slice(ps, func(i, k int) bool {
		if ps[i].Lang != ps[k].Lang {
			return ps[i].Lang < ps[k].Lang
		}
		return ps[i].Path < ps[k].Path
	})
	var b strings.Builder
	b.WriteString(proposalsHeader)
	lang := ""
	for _, p := range ps {
		if p.Lang != lang {
			lang = p.Lang
			fmt.Fprintf(&b, "\n## %s (%s)\n\n", languages[lang], lang)
		}
		tick := " "
		if p.Ticked {
			tick = "x"
		}
		fmt.Fprintf(&b, "- [%s] `%s`\n  - now: %s\n  - proposed: %s\n  - why: %s\n",
			tick, p.Path, quote(p.Now), quote(p.Proposed), oneLine(p.Reason))
	}
	writeFile(proposalsFile, b.String())
}

// addProposals adds ps to the checklist, keeping what it holds and its ticks.
func addProposals(ps []proposal) {
	all, _ := readProposals()
	seen := map[string]bool{}
	for _, p := range all {
		seen[p.key()] = true
	}
	for _, p := range ps {
		if !seen[p.key()] {
			all = append(all, p)
			seen[p.key()] = true
		}
	}
	if len(all) > 0 {
		writeProposals(all)
	}
}

var (
	headingRE = regexp.MustCompile(`^## .*\(([a-z]+)\)\s*$`)
	itemRE    = regexp.MustCompile("^- \\[( |x|X)\\] `([^`]+)`\\s*$")
	detailRE  = regexp.MustCompile(`^\s+- (now|proposed|why): (.*)$`)
)

// readProposals reads the checklist; none without the file.
func readProposals() ([]proposal, error) {
	f, err := os.Open(proposalsFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var (
		ps   []proposal
		lang string
		cur  *proposal
	)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if m := headingRE.FindStringSubmatch(line); m != nil {
			lang = m[1]
			continue
		}
		if m := itemRE.FindStringSubmatch(line); m != nil {
			ps = append(ps, proposal{Lang: lang, Path: m[2], Ticked: m[1] != " "})
			cur = &ps[len(ps)-1]
			continue
		}
		if m := detailRE.FindStringSubmatch(line); m != nil && cur != nil {
			switch m[1] {
			case "now", "proposed":
				var s string
				if err := json.Unmarshal([]byte(m[2]), &s); err != nil {
					return nil, fmt.Errorf("%s:%d: %s is not a quoted text: %v", proposalsFile, n, m[1], err)
				}
				if m[1] == "now" {
					cur.Now = s
				} else {
					cur.Proposed = s
				}
			case "why":
				cur.Reason = m[2]
			}
		}
	}
	return ps, sc.Err()
}

// applyProposals applies the ticked proposals whose text has not changed
// since and whose placeholders match the English, then removes the
// checklist.
func applyProposals(en string) {
	ps, err := readProposals()
	if err != nil {
		fail("%v", err)
	}
	if ps == nil {
		fmt.Println("No proposals to apply (" + proposalsFile + " does not exist).")
		return
	}
	enLeaves, err := leaves(en)
	if err != nil {
		fail("en.json: %v", err)
	}
	byLang := map[string]map[string]string{}
	applied, skipped := 0, 0
	for _, p := range ps {
		if !p.Ticked {
			continue
		}
		path := filepath.Join(localesDir, p.Lang+".json")
		b, err := os.ReadFile(path)
		if err != nil {
			fail("%s: %v", p.Lang, err)
		}
		current, err := leaves(string(b))
		if err != nil {
			fail("%s: %v", p.Lang, err)
		}
		if now, ok := current[p.Path]; !ok || now != p.Now {
			fmt.Fprintf(os.Stderr, "  skipped %s %s: the text changed since it was proposed\n", p.Lang, p.Path)
			skipped++
			continue
		}
		if problems := checkTexts(map[string]string{p.Path: enLeaves[p.Path]}, map[string]string{p.Path: p.Proposed}); len(problems) > 0 {
			fmt.Fprintf(os.Stderr, "  skipped %s %s: %s\n", p.Lang, p.Path, strings.Join(problems, "; "))
			skipped++
			continue
		}
		if byLang[p.Lang] == nil {
			byLang[p.Lang] = map[string]string{}
		}
		byLang[p.Lang][p.Path] = p.Proposed
		applied++
	}
	for lang, values := range byLang {
		path := filepath.Join(localesDir, lang+".json")
		b, _ := os.ReadFile(path)
		out, err := fillPaths(string(b), values)
		if err != nil {
			fail("%s: %v", lang, err)
		}
		writeFile(path, out)
	}
	mirrorToServer()
	if err := os.Remove(proposalsFile); err != nil {
		fail("%v", err)
	}
	unticked := 0
	for _, p := range ps {
		if !p.Ticked {
			unticked++
		}
	}
	fmt.Printf("✓ Applied %d proposal(s), skipped %d, dropped %d unticked.\n", applied, skipped, unticked)
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
