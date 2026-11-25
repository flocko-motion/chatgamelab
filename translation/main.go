// Command translation translates the platform's UI texts from en.json and
// de.json into every other language, with OpenAI or with the local Claude
// Code CLI.
//
// Run it from the repository root:
//
//	go run ./translation                  translate what changed, with OpenAI
//	go run ./translation --engine claude  the same with Claude Code, which also
//	                                      proposes improvements to existing texts
//	go run ./translation --apply          apply the proposals ticked in
//	                                      translation/proposals.md
//	go run ./translation --accept         check the fields translated by hand (or by
//	                                      an agent) and record them as translated
//	go run ./translation --check          fail if anything needs translating or a
//	                                      translation is broken; calls no AI
//
// Each target language keeps a <lang>.hash.json next to its <lang>.json. It
// holds a hash of the en + de text of every field, so a run sends only the
// fields whose source text changed since that language was last translated.
// Delete a hash file to translate that language from scratch. A field en.json
// no longer has is dropped from every language.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// The settings, fixed here so every run translates the same way.
const (
	// Where en.json and de.json are written by hand, and every translation lands.
	localesDir = "frontend/src/i18n/locales"
	// The server embeds a copy of the locales; it mirrors localesDir.
	serverLocalesDir = "go/lang/locales"

	model       = "gpt-6-luna"
	apiKeyEnv   = "OPENAI_API_KEY"
	attempts    = 3
	placeholder = "TODO: TRANSLATE"

	instruction = "You are an expert in translation of json structured language files for games. " +
		"Translate the given JSON object to the target language while preserving the exact same structure and keys. " +
		"Only translate the string values. Return a valid JSON object. " +
		"You get the original already in two languages, so that you have more context to understand the intention of each field."
)

var sources = []string{"en", "de"}

// languages are the UI languages, by code and native name. languages.json is
// written from this list, and the server reads its languages from that file.
var languages = map[string]string{
	"en":  "English",
	"de":  "Deutsch",
	"fr":  "Français",
	"es":  "Español",
	"it":  "Italiano",
	"pt":  "Português",
	"nl":  "Nederlands",
	"pl":  "Polski",
	"ru":  "Русский",
	"ja":  "日本語",
	"ko":  "한국어",
	"zh":  "中文",
	"ar":  "العربية",
	"hi":  "हिन्दी",
	"tr":  "Türkçe",
	"sv":  "Svenska",
	"da":  "Dansk",
	"no":  "Norsk",
	"fi":  "Suomi",
	"so":  "Soomaali",
	"ps":  "پښتو",
	"fa":  "فارسی",
	"uk":  "Українська",
	"bar": "Bayrisch",
	"el":  "Ελληνικά",
	"sr":  "Српски",
	"bs":  "Bosanski",
	"sq":  "Shqip",
	"bg":  "Български",
	"hu":  "Magyar",
	"hr":  "Hrvatski",
	"sl":  "Slovenščina",
	"cs":  "Čeština",
	"sk":  "Slovenčina",
	"ro":  "Română",
	"ti":  "ትግርኛ",
	"id":  "Bahasa Indonesia",
}

func main() {
	check := flag.Bool("check", false, "fail if anything needs translating; calls no AI and writes nothing")
	engine := flag.String("engine", "openai", "who translates: openai (OPENAI_API_KEY) or claude (the local Claude Code CLI, signed in)")
	apply := flag.Bool("apply", false, "apply the proposals ticked in "+proposalsFile)
	accept := flag.Bool("accept", false, "check the fields that need translating, translated in the files by hand, and record them as translated")
	flag.Parse()

	if _, err := os.Stat(localesDir); err != nil {
		fail("%s not found — run this from the repository root", localesDir)
	}

	src := readSources(*check)
	if *apply {
		applyProposals(src[0])
		return
	}
	pruned := prune(src[0], *check)
	hashes, err := fieldHashes(src...)
	if err != nil {
		fail("hashing the sources: %v", err)
	}

	if *accept {
		acceptAll(src[0], pending(src, hashes), hashes)
		return
	}

	todo := pending(src, hashes)
	stale := languagesJSONStale() || serverCopyStale()
	broken := checkAll(src[0])

	if *check {
		if len(todo) > 0 || stale || pruned || broken {
			for _, j := range todo {
				fmt.Fprintf(os.Stderr, "✗ %s (%s): %d fields need translating\n", languages[j.lang], j.lang, j.changed)
			}
			if stale {
				fmt.Fprintf(os.Stderr, "✗ languages.json or the server's copy of the locales is out of date\n")
			}
			if pruned {
				fmt.Fprintf(os.Stderr, "✗ some languages hold fields en.json no longer has\n")
			}
			if broken {
				fmt.Fprintf(os.Stderr, "✗ some translations lack a field or change its placeholders\n")
			}
			fail("run make translate")
		}
		fmt.Println("✓ Translations are complete")
		return
	}

	if len(todo) > 0 {
		var tr translator
		switch *engine {
		case "openai":
			apiKey := os.Getenv(apiKeyEnv)
			if apiKey == "" {
				fail("%s is not set", apiKeyEnv)
			}
			tr = openAITranslator(apiKey)
		case "claude":
			if !claudeAvailable() {
				fail("the Claude Code CLI is not installed or not signed in (claude auth status)")
			}
			tr = claudeTranslator(src)
		default:
			fail("unknown engine %q: openai or claude", *engine)
		}
		translateAll(tr, todo, hashes)
	} else {
		fmt.Println("✓ Every language is up to date")
	}

	writeLanguagesJSON()
	mirrorToServer()
}

// readSources reads en.json and de.json. They must have the same fields and no
// placeholders left: a field present in one only is added to the other as a
// placeholder, for whoever is translating to fill in.
func readSources(check bool) []string {
	src := make([]string, len(sources))
	for i, code := range sources {
		b, err := os.ReadFile(filepath.Join(localesDir, code+".json"))
		if err != nil {
			fail("%v", err)
		}
		src[i] = string(b)
	}

	patched0, patched1, added0, added1, err := syncStructures(src[0], src[1], placeholder)
	if err != nil {
		fail("comparing en.json and de.json: %v", err)
	}
	if len(added0) > 0 || len(added1) > 0 {
		report := func(code string, added []string) {
			for _, f := range added {
				fmt.Fprintf(os.Stderr, "  %s.json + %s\n", code, f)
			}
		}
		fmt.Fprintf(os.Stderr, "en.json and de.json have different fields:\n")
		report("en", added0)
		report("de", added1)
		if check {
			fail("run make translate to add them as %q", placeholder)
		}
		writeFile(filepath.Join(localesDir, "en.json"), patched0)
		writeFile(filepath.Join(localesDir, "de.json"), patched1)
		fail("added them as %q — fill them in and run again", placeholder)
	}

	missing := false
	for i, code := range sources {
		for _, f := range findPlaceholders(src[i], placeholder) {
			fmt.Fprintf(os.Stderr, "  %s.json %s\n", code, f)
			missing = true
		}
	}
	if missing {
		fail("fill in the %q fields above and run again", placeholder)
	}
	return src
}

// prune drops from every target language the fields en.json no longer has,
// and says whether there were any; with check it only reports them.
func prune(en string, check bool) bool {
	found := false
	for _, code := range targetLanguages() {
		path := filepath.Join(localesDir, code+".json")
		b, err := os.ReadFile(path)
		if err != nil {
			continue // a new language: nothing to drop
		}
		out, dropped, err := pruneFields(string(b), en)
		if err != nil {
			fail("%s: %v", code, err)
		}
		if len(dropped) == 0 {
			continue
		}
		found = true
		for _, f := range dropped {
			fmt.Fprintf(os.Stderr, "  %s.json - %s\n", code, f)
		}
		if !check {
			writeFile(path, out)
		}
	}
	return found
}

type job struct {
	lang     string
	existing string          // the language's current file, empty for a new language
	changed  int             // how many fields to translate
	inputs   []string        // the changed fields, once per source language
	paths    map[string]bool // the changed fields' paths
}

// pending lists the languages with fields whose source text changed since they
// were last translated.
func pending(src []string, hashes map[string]string) []job {
	var jobs []job
	for _, code := range targetLanguages() {
		var existing string
		if b, err := os.ReadFile(filepath.Join(localesDir, code+".json")); err == nil {
			existing = string(b)
		}
		old := map[string]string{}
		if b, err := os.ReadFile(filepath.Join(localesDir, code+".hash.json")); err == nil {
			_ = json.Unmarshal(b, &old)
		}

		have := map[string]string{}
		if existing != "" {
			if l, err := leaves(existing); err == nil {
				have = l
			}
		}

		// a field changed since last translated, or missing from the language
		changed := map[string]bool{}
		for path, h := range hashes {
			if _, ok := have[path]; old[path] != h || !ok {
				changed[path] = true
			}
		}
		if len(changed) == 0 {
			continue
		}

		j := job{lang: code, existing: existing, changed: len(changed), paths: changed}
		for _, s := range src {
			part, err := extractPaths(s, changed)
			if err != nil {
				fail("%s: %v", code, err)
			}
			j.inputs = append(j.inputs, part)
		}
		jobs = append(jobs, j)
	}
	return jobs
}

// translator translates one language's job: the changed fields as a JSON
// tree, and any proposals for its existing texts. warn tells of an attempt
// that failed.
type translator struct {
	name     string
	parallel int // languages at once; 0: all
	run      func(j job, warn func(string, ...any)) (string, []proposal, usage, error)
}

// openAITranslator translates with OpenAI, a few attempts per language.
func openAITranslator(apiKey string) translator {
	return translator{name: model, run: func(j job, warn func(string, ...any)) (string, []proposal, usage, error) {
		var total usage
		var err error
		for attempt := 1; attempt <= attempts; attempt++ {
			var out string
			var u usage
			out, u, err = translate(context.Background(), apiKey, j.inputs, languages[j.lang])
			total = total.add(u)
			if err == nil {
				return out, nil, total, nil
			}
			warn("attempt %d/%d failed: %v", attempt, attempts, err)
		}
		return "", nil, total, err
	}}
}

// claudeTranslator translates with the Claude Code CLI: the tool checks each
// answer and hands what is wrong back to Claude, a few rounds per language.
func claudeTranslator(src []string) translator {
	en, err := leaves(src[0])
	if err != nil {
		fail("en.json: %v", err)
	}
	de, err := leaves(src[1])
	if err != nil {
		fail("de.json: %v", err)
	}
	return translator{name: "Claude Code (" + claudeModel + ")", parallel: claudeParallel,
		run: func(j job, warn func(string, ...any)) (string, []proposal, usage, error) {
			fields := map[string]field{}
			want := map[string]string{}
			for p := range j.paths {
				fields[p] = field{EN: en[p], DE: de[p]}
				want[p] = en[p]
			}
			existing := map[string]string{}
			if j.existing != "" {
				existing, _ = leaves(j.existing)
			}
			var total usage
			var problems []string
			for attempt := 1; attempt <= attempts; attempt++ {
				answer, u, err := askClaude(context.Background(), languages[j.lang], fields, existing, problems)
				total = total.add(u)
				if err != nil {
					warn("attempt %d/%d failed: %v", attempt, attempts, err)
					continue
				}
				if problems = checkTexts(want, answer.Translations); len(problems) > 0 {
					warn("attempt %d/%d: %d problem(s), asking again", attempt, attempts, len(problems))
					continue
				}
				part, err := extractPaths(src[0], j.paths)
				if err != nil {
					return "", nil, total, err
				}
				out, err := fillPaths(part, answer.Translations)
				return out, proposalsOf(j.lang, answer, existing, en), total, err
			}
			if len(problems) > 0 {
				return "", nil, total, fmt.Errorf("still wrong after %d attempts: %s", attempts, strings.Join(problems, "; "))
			}
			return "", nil, total, fmt.Errorf("no answer after %d attempts", attempts)
		}}
}

// proposalsOf are the answer's proposals worth a person's look: for an
// existing text, changing it, with a reason and the English placeholders.
func proposalsOf(lang string, a claudeAnswer, existing, en map[string]string) []proposal {
	var out []proposal
	for _, p := range a.Proposals {
		now, ok := existing[p.Path]
		if !ok || p.Proposed == now || strings.TrimSpace(p.Reason) == "" {
			continue
		}
		if len(checkTexts(map[string]string{p.Path: en[p.Path]}, map[string]string{p.Path: p.Proposed})) > 0 {
			continue
		}
		out = append(out, proposal{Lang: lang, Path: p.Path, Now: now, Proposed: p.Proposed, Reason: p.Reason})
		if len(out) == maxProposals {
			break
		}
	}
	return out
}

// translateAll translates the pending languages, at once or as many as the
// translator takes, and writes each language as soon as it is done.
func translateAll(tr translator, jobs []job, hashes map[string]string) {
	fmt.Printf("Translating %d language(s) with %s...\n", len(jobs), tr.name)

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		total     usage
		failed    int
		proposals []proposal
	)
	log := func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Printf(format+"\n", args...)
	}
	parallel := tr.parallel
	if parallel == 0 {
		parallel = len(jobs)
	}
	slots := make(chan struct{}, parallel)

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			name := languages[j.lang]
			warn := func(format string, args ...any) {
				log("⚠ %s (%s): "+format, append([]any{name, j.lang}, args...)...)
			}

			out, ps, u, err := tr.run(j, warn)
			mu.Lock()
			total = total.add(u)
			proposals = append(proposals, ps...)
			mu.Unlock()
			if err == nil && j.existing != "" {
				out, err = mergeJSON(j.existing, out)
			}
			if err == nil {
				out, err = indentJSON(out)
			}
			if err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				log("✗ %s (%s): %v", name, j.lang, err)
				return
			}

			writeFile(filepath.Join(localesDir, j.lang+".json"), out)
			writeFile(filepath.Join(localesDir, j.lang+".hash.json"), mustIndent(hashes))
			log("✓ %s (%s): %d fields", name, j.lang, j.changed)
		}(j)
	}
	wg.Wait()

	fmt.Printf("Tokens: %d in, %d out, %d total\n", total.Input, total.Output, total.Total)
	if len(proposals) > 0 {
		addProposals(proposals)
		fmt.Printf("✎ %d proposal(s) to improve existing texts: tick them in %s, then run make translate-apply\n", len(proposals), proposalsFile)
	}
	if failed > 0 {
		fail("%d language(s) failed — run again to retry them", failed)
	}
}

func targetLanguages() []string {
	var codes []string
	for code := range languages {
		if code != sources[0] && code != sources[1] {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	return codes
}

// languagesJSON is the language list as the web app and the server read it.
func languagesJSON() string {
	type language struct {
		Code  string `json:"code"`
		Label string `json:"label"`
	}
	list := make([]language, 0, len(languages))
	for code, label := range languages {
		list = append(list, language{code, label})
	}
	sort.Slice(list, func(i, k int) bool { return list[i].Code < list[k].Code })
	return mustIndent(list)
}

func writeLanguagesJSON() {
	writeFile(filepath.Join(localesDir, "languages.json"), languagesJSON())
}

func languagesJSONStale() bool {
	b, err := os.ReadFile(filepath.Join(localesDir, "languages.json"))
	return err != nil || string(b) != languagesJSON()
}

// mirrorToServer replaces the server's copy of the locales with localesDir.
func mirrorToServer() {
	old, _ := filepath.Glob(filepath.Join(serverLocalesDir, "*.json"))
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			fail("%v", err)
		}
	}
	files, _ := filepath.Glob(filepath.Join(localesDir, "*.json"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			fail("%v", err)
		}
		writeFile(filepath.Join(serverLocalesDir, filepath.Base(f)), string(b))
	}
}

func serverCopyStale() bool {
	want, _ := filepath.Glob(filepath.Join(localesDir, "*.json"))
	have, _ := filepath.Glob(filepath.Join(serverLocalesDir, "*.json"))
	if len(want) != len(have) {
		return true
	}
	for _, f := range want {
		a, _ := os.ReadFile(f)
		b, err := os.ReadFile(filepath.Join(serverLocalesDir, filepath.Base(f)))
		if err != nil || string(a) != string(b) {
			return true
		}
	}
	return false
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fail("%v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	os.Exit(1)
}

// checkAll checks every language's texts against en.json: each with the
// English placeholders. It reports what is wrong and says whether anything
// is.
func checkAll(en string) bool {
	want, err := leaves(en)
	if err != nil {
		fail("en.json: %v", err)
	}
	broken := false
	for _, code := range targetLanguages() {
		b, err := os.ReadFile(filepath.Join(localesDir, code+".json"))
		if err != nil {
			continue // a new language: translated by the next run
		}
		have, err := leaves(string(b))
		if err != nil {
			fail("%s: %v", code, err)
		}
		asked := map[string]string{}
		for path := range have {
			if en, ok := want[path]; ok {
				asked[path] = en // a field missing is pending; one en lacks, prune's
			}
		}
		for _, p := range checkTexts(asked, have) {
			fmt.Fprintf(os.Stderr, "  %s.json %s\n", code, p)
			broken = true
		}
	}
	return broken
}

// acceptAll records the fields of jobs as translated, after checking the
// texts written for them in each language's file: each there, with the
// English placeholders. Nothing is recorded while any is wrong.
func acceptAll(en string, jobs []job, hashes map[string]string) {
	want, err := leaves(en)
	if err != nil {
		fail("en.json: %v", err)
	}
	wrong := 0
	for _, j := range jobs {
		have, err := leaves(j.existing)
		if err != nil {
			fail("%s: %v", j.lang, err)
		}
		asked, got := map[string]string{}, map[string]string{}
		for p := range j.paths {
			asked[p] = want[p]
			if v, ok := have[p]; ok {
				got[p] = v
			}
		}
		for _, p := range checkTexts(asked, got) {
			fmt.Fprintf(os.Stderr, "  %s.json %s\n", j.lang, p)
			wrong++
		}
	}
	if wrong > 0 {
		fail("%d field(s) not translated right — fix them and run again", wrong)
	}
	// texts written by hand come in any key order; the tool's is sorted
	for _, code := range targetLanguages() {
		path := filepath.Join(localesDir, code+".json")
		if b, err := os.ReadFile(path); err == nil {
			if out, err := indentJSON(string(b)); err == nil && out != string(b) {
				writeFile(path, out)
			}
		}
	}
	for _, j := range jobs {
		writeFile(filepath.Join(localesDir, j.lang+".hash.json"), mustIndent(hashes))
		fmt.Printf("✓ %s (%s): %d fields accepted\n", languages[j.lang], j.lang, j.changed)
	}
	writeLanguagesJSON()
	mirrorToServer()
	if len(jobs) == 0 {
		fmt.Println("✓ Every language is up to date")
	}
}
