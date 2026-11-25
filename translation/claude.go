package main

// The Claude engine translates with the local Claude Code CLI (`claude -p`),
// on its user's login: no API key. Besides the fields asked for, Claude may
// propose improvements to a language's existing texts, each with its reason;
// they are only proposed (proposals.go), never applied unasked.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

const (
	claudeModel = "sonnet"
	// claudeParallel is how many languages Claude translates at once.
	claudeParallel = 4
	// maxProposals bounds a language's proposals per run, so the list stays
	// one a person reads.
	maxProposals = 10
)

const claudeInstruction = "You translate the UI texts of ChatGameLab, a platform where young people and educators " +
	"write and play AI text adventures. You get each field in English and German (both written by hand: " +
	"together they show the intention), and the target language's current texts for context. " +
	"Keep every {{placeholder}} exactly as it is. Match the tone and the terms the target language already uses. " +
	"Besides translating the fields asked for, you may propose improvements to existing texts of the target " +
	"language, only for real mistakes: a wrong meaning, a wrong or inconsistent term, broken grammar, or a text " +
	"still in English. Not for taste. Give each proposal's reason in English, so a reviewer who does not speak " +
	"the language can judge it."

// claudeSchema is the answer Claude gives: translations and proposals.
const claudeSchema = `{"type":"object","properties":{
"translations":{"type":"object","additionalProperties":{"type":"string"}},
"proposals":{"type":"array","items":{"type":"object","properties":{
  "path":{"type":"string"},"proposed":{"type":"string"},"reason":{"type":"string"}},
  "required":["path","proposed","reason"]}}},
"required":["translations","proposals"]}`

// claudeAnswer is what Claude answers for one language.
type claudeAnswer struct {
	Translations map[string]string `json:"translations"`
	Proposals    []struct {
		Path     string `json:"path"`
		Proposed string `json:"proposed"`
		Reason   string `json:"reason"`
	} `json:"proposals"`
}

// claudeAvailable says whether the Claude Code CLI is installed and signed in.
func claudeAvailable() bool {
	if _, err := exec.LookPath("claude"); err != nil {
		return false
	}
	out, err := exec.Command("claude", "auth", "status").CombinedOutput()
	return err == nil && !strings.Contains(strings.ToLower(string(out)), "not logged in")
}

// field is one field to translate, in both source languages.
type field struct {
	EN, DE string
}

// askClaude has Claude translate fields (path → sources) into language, with
// existing (path → text) as context; problems are what the tool found wrong
// with the last answer, for Claude to fix.
func askClaude(ctx context.Context, language string, fields map[string]field, existing map[string]string, problems []string) (claudeAnswer, usage, error) {
	paths := make([]string, 0, len(fields))
	for p := range fields {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	fmt.Fprintf(&b, "Target language: %s\n\n", language)
	b.WriteString("Translate these fields (path, English, German):\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "- %s\n  en: %q\n  de: %q\n", p, fields[p].EN, fields[p].DE)
	}
	if len(existing) > 0 {
		fmt.Fprintf(&b, "\nThe language's current texts (path: text), for context and for proposals (at most %d):\n", maxProposals)
		keys := make([]string, 0, len(existing))
		for p := range existing {
			keys = append(keys, p)
		}
		sort.Strings(keys)
		for _, p := range keys {
			fmt.Fprintf(&b, "%s: %q\n", p, existing[p])
		}
	}
	if len(problems) > 0 {
		b.WriteString("\nYour last answer had these problems; fix them:\n- " + strings.Join(problems, "\n- ") + "\n")
	}

	cmd := exec.CommandContext(ctx, "claude", "-p",
		"--model", claudeModel,
		"--output-format", "json",
		"--json-schema", claudeSchema,
		"--system-prompt", claudeInstruction,
		"--tools", "",
		"--no-session-persistence")
	cmd.Stdin = strings.NewReader(b.String())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return claudeAnswer{}, usage{}, fmt.Errorf("claude: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var r struct {
		IsError          bool         `json:"is_error"`
		Result           string       `json:"result"`
		StructuredOutput claudeAnswer `json:"structured_output"`
		Usage            struct {
			Input   int `json:"input_tokens"`
			Cached  int `json:"cache_read_input_tokens"`
			Written int `json:"cache_creation_input_tokens"`
			Output  int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return claudeAnswer{}, usage{}, fmt.Errorf("claude's answer: %w", err)
	}
	u := usage{Input: r.Usage.Input + r.Usage.Cached + r.Usage.Written, Output: r.Usage.Output}
	u.Total = u.Input + u.Output
	if r.IsError {
		return claudeAnswer{}, u, fmt.Errorf("claude: %s", r.Result)
	}
	return r.StructuredOutput, u, nil
}
