// Command models prints the engine's catalogue of AI platforms.
//
//	models          # as text
//	models -json    # as the engine's HTTP handler serves it
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"engine"
)

func main() {
	asJSON := flag.Bool("json", false, "print JSON, as GET platforms serves it")
	flag.Parse()

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(engine.Platforms()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)
	for i, p := range engine.Platforms() {
		if i > 0 {
			fmt.Fprintln(w)
		}
		describe(w, p)
	}
	_ = w.Flush()

	// Aligning the parameters pads the lines that have none.
	for line := range strings.Lines(out.String()) {
		fmt.Println(strings.TrimRight(line, " \n"))
	}
}

func describe(w *tabwriter.Writer, p engine.PlatformInfo) {
	keys := "no"
	if p.UserKeys {
		keys = "yes"
	}
	fmt.Fprintf(w, "%s (%s)\n", p.Name, p.ID)
	fmt.Fprintf(w, "  user keys\t%s\n", keys)
	fmt.Fprintf(w, "  toggles\t%s\n", toggles(p.Toggles))
	fmt.Fprintf(w, "  genres\t%s\n", join(p.Genres))

	types := p.Toggles.ModelTypes()
	for _, tier := range p.Tiers {
		fmt.Fprintf(w, "  %s\t%s\n", tier.Tier, tier.Label)
		for _, kind := range types {
			model := tier.Models[kind]
			fmt.Fprintf(w, "    %s\t%s\t%s\n", kind, model.Name, params(model.Params))
		}
	}
}

func toggles(t engine.Toggles) string {
	var parts []string
	for _, toggle := range []engine.Toggle{engine.ToggleImages, engine.ToggleAudioOut, engine.ToggleAudioIn, engine.ToggleAudioLive} {
		mark := "-"
		if t.Has(toggle) {
			mark = "+"
		}
		parts = append(parts, mark+string(toggle))
	}
	return strings.Join(parts, " ")
}

func params(p map[string]string) string {
	var parts []string
	for _, k := range slices.Sorted(maps.Keys(p)) {
		parts = append(parts, k+"="+p[k])
	}
	return strings.Join(parts, " ")
}

func join[T ~string](items []T) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = string(item)
	}
	return strings.Join(parts, " ")
}
