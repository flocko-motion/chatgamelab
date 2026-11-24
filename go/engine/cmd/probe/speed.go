package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/devconfig"
	"engine/internal/genre"
)

// speedProse is the plain turn the speed benchmark asks for: narration of a
// game's length, as Expand writes it.
const speedProse = "NARRATE: The player crosses the rope bridge in the storm; halfway over, a plank breaks. Write 3 to 6 sentences of atmospheric prose, ending with what the player sees ahead."

// speedRow is what the rounds of one model and call kind brought.
type speedRow struct {
	platform, model, call string
	first, total          []time.Duration // to the first piece of text, and to the whole reply
	rates                 []float64       // output tokens per second, after the first piece
	outputTokens          []int64
	errored               int
	cost                  float64
	priced                bool
	lastError             string
}

// probeSpeed times every text model of a platform's model table: a prose
// turn and the structured Outline turn, each on a fresh thread, as a game's
// first turn would be. It reports the time to the first piece of text, the
// time to the whole reply, the output tokens per second and the cost.
func probeSpeed(only string, rounds int) {
	schema := genre.OutlineSchema(status)
	adventure, err := genre.NewAdventure(genre.AdventureConfig{Tool: mock.Tool{}, Hearing: mock.Hearing{}, Thread: mock.Thread{}, Image: mock.Image{}, Status: status})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	instructions := adventure.Prompts[genre.PromptGameMaster]
	outline, _ := json.Marshal(map[string]any{
		"input":  "The player tries to cross the rope bridge in the storm, hoping it holds.",
		"status": status,
	})

	var rows []speedRow
	for _, p := range platforms() {
		if only != "" && only != p.id {
			continue
		}
		key, err := devconfig.APIKey(p.id)
		if err != nil || key == "" {
			fmt.Fprintf(os.Stderr, "%s: skipped, no key\n", p.id)
			continue
		}
		keys := func(context.Context) (string, error) { return key, nil }
		for _, model := range textModels(p) {
			fmt.Fprintf(os.Stderr, "%s %s: %d rounds\n", p.id, model, rounds)
			prose := speedRow{platform: p.id, model: model, call: "prose"}
			structured := speedRow{platform: p.id, model: model, call: "outline"}
			for range rounds {
				prose.time(p, keys, model, adapters.ThreadTurn{System: instructions, Directive: true, Input: speedProse})
				structured.time(p, keys, model, adapters.ThreadTurn{System: instructions, Input: string(outline), Schema: &schema})
			}
			rows = append(rows, prose, structured)
		}
	}
	reportSpeed(rows)
}

// textModels are the models of a platform's model table the probe can time
// on a thread: those that answer in text, but not the hearing and live
// models, which take no text turn.
func textModels(p platform) []string {
	var out []string
	for name, facts := range p.prices {
		if !facts.Takes(adapters.MediumText) || !facts.Makes(adapters.MediumText) {
			continue
		}
		if strings.HasPrefix(name, "gpt-audio") || strings.HasPrefix(name, "gpt-live") || strings.HasPrefix(name, "voxtral") {
			continue
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// time makes one turn on a fresh thread and records it.
func (r *speedRow) time(p platform, keys adapters.KeyFunc, model string, turn adapters.ThreadTurn) {
	thread := p.thread(keys, model)
	defer closeThread(thread)
	began := time.Now()
	var first time.Duration
	turn.Piece = func(string) {
		if first == 0 {
			first = time.Since(began)
		}
	}
	reply, err := thread.Continue(context.Background(), turn)
	total := time.Since(began)
	if price, ok := p.prices.Price(model); ok {
		r.cost += price.Cost(reply.Usage)
		r.priced = true
	}
	if err != nil {
		r.errored++
		r.lastError = err.Error()
		return
	}
	if first == 0 {
		// Nothing streamed: the whole reply came at once.
		first = total
	}
	r.first = append(r.first, first)
	r.total = append(r.total, total)
	r.outputTokens = append(r.outputTokens, reply.Usage.OutputTokens)
	if streamed := total - first; streamed > 0 && reply.Usage.OutputTokens > 0 {
		r.rates = append(r.rates, float64(reply.Usage.OutputTokens)/streamed.Seconds())
	}
}

func reportSpeed(rows []speedRow) {
	// Fastest whole reply first, per call kind.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].call != rows[j].call {
			return rows[i].call > rows[j].call
		}
		return medianOf(rows[i].total) < medianOf(rows[j].total)
	})
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "call\tplatform\tmodel\tfirst text\twhole reply\ttokens/s\toutput tokens\terrored\tper call\t")
	for _, r := range rows {
		cost := "unpriced"
		if n := len(r.total) + r.errored; r.priced && n > 0 {
			cost = fmt.Sprintf("$%.5f", r.cost/float64(n))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t\n",
			r.call, r.platform, r.model, median(r.first), median(r.total), medianRate(r.rates), medianTokens(r.outputTokens), r.errored, cost)
	}
	_ = w.Flush()
	for _, r := range rows {
		if r.lastError != "" {
			fmt.Printf("\n%s %s %s last problem: %s\n", r.call, r.platform, r.model, strings.TrimSpace(r.lastError))
		}
	}
}

func medianOf(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return time.Duration(1<<62 - 1)
	}
	s := slices.Clone(ds)
	slices.Sort(s)
	return s[len(s)/2]
}

func medianRate(rs []float64) string {
	if len(rs) == 0 {
		return "-"
	}
	s := slices.Clone(rs)
	slices.Sort(s)
	return fmt.Sprintf("%.0f", s[len(s)/2])
}
