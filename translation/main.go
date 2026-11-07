// Command translation translates the platform's UI texts from en.json and
// de.json into every other language with OpenAI.
//
// Run it from the repository root:
//
//	go run ./translation           translate what changed
//	go run ./translation --check   fail if anything needs translating; calls no AI
//
// Each target language keeps a <lang>.hash.json next to its <lang>.json. It
// holds a hash of the en + de text of every field, so a run sends only the
// fields whose source text changed since that language was last translated.
// Delete a hash file to translate that language from scratch.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// The settings, fixed here so every run translates the same way.
const (
	// Where en.json and de.json are written by hand, and every translation lands.
	localesDir = "platform/web/src/i18n/locales"
	// The server embeds a copy of the locales; it mirrors localesDir.
	serverLocalesDir = "platform/server/lang/locales"

	model       = "gpt-5.6-luna"
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
	flag.Parse()

	if _, err := os.Stat(localesDir); err != nil {
		fail("%s not found — run this from the repository root", localesDir)
	}

	src := readSources(*check)
	hashes, err := fieldHashes(src...)
	if err != nil {
		fail("hashing the sources: %v", err)
	}

	todo := pending(src, hashes)
	stale := languagesJSONStale() || serverCopyStale()

	if *check {
		if len(todo) > 0 || stale {
			for _, j := range todo {
				fmt.Fprintf(os.Stderr, "✗ %s (%s): %d fields need translating\n", languages[j.lang], j.lang, j.changed)
			}
			if stale {
				fmt.Fprintf(os.Stderr, "✗ languages.json or the server's copy of the locales is out of date\n")
			}
			fail("run make translate")
		}
		fmt.Println("✓ Translations are complete")
		return
	}

	if len(todo) > 0 {
		apiKey := os.Getenv(apiKeyEnv)
		if apiKey == "" {
			fail("%s is not set", apiKeyEnv)
		}
		translateAll(apiKey, todo, hashes)
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

type job struct {
	lang     string
	existing string   // the language's current file, empty for a new language
	changed  int      // how many fields to translate
	inputs   []string // the changed fields, once per source language
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

		changed := map[string]bool{}
		for path, h := range hashes {
			if old[path] != h {
				changed[path] = true
			}
		}
		if len(changed) == 0 {
			continue
		}

		j := job{lang: code, existing: existing, changed: len(changed)}
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

// translateAll translates every pending language at once, each with a few
// attempts, and writes each language as soon as it is done.
func translateAll(apiKey string, jobs []job, hashes map[string]string) {
	fmt.Printf("Translating %d language(s) with %s...\n", len(jobs), model)

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		total  usage
		failed int
	)
	log := func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Printf(format+"\n", args...)
	}

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			name := languages[j.lang]

			var out string
			var err error
			for attempt := 1; attempt <= attempts; attempt++ {
				var u usage
				out, u, err = translate(context.Background(), apiKey, j.inputs, name)
				mu.Lock()
				total = total.add(u)
				mu.Unlock()
				if err == nil {
					break
				}
				log("⚠ %s (%s): attempt %d/%d failed: %v", name, j.lang, attempt, attempts, err)
			}
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
