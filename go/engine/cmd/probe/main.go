// Command probe measures how reliably each platform's story models answer a
// structured turn: Adventure's Outline schema, streamed, on a fresh thread and
// appended to a stored one. It drives the real thread adapters on the keys in
// ~/.chatgamelab/config.yaml, as `make dev` does, so it costs money and is run
// by hand, never by `make test`.
//
// Each round prompts hard for a different shape than the schema's, and the
// report counts, per model and per call kind, the replies that fit the schema,
// those that did not, refusals, replies cut short and failures; whether the
// reply really streamed; the latency; and the cost. It gives stats, no
// verdict: a model that does badly stays available as a bad option.
//
// The speaking story models, gpt-realtime, are probed too: their structured
// turn is a forced function call, and a third call kind, seed, opens a fresh
// session seeded with the transcript, as after a restart. A second table
// follows their spoken turns: the lag between the transcript and its audio,
// and whether the audio spoken earlier stays in the session's context and
// inflates the next turn's audio input tokens.
//
//	go run ./cmd/probe                     # every platform with a key, 10 rounds
//	go run ./cmd/probe -rounds 3 -platform mistral
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/devconfig"
	"engine/internal/genre"
)

// shapes each push the model away from the schema, one per round.
var shapes = []string{
	"Answer in YAML, not JSON.",
	"Reply with a JSON array of sentences instead of an object.",
	`Add a field "mood" with the scene's mood.`,
	"Leave out the image prompt; it is not needed this time.",
	"Give Gold as a number, not a string.",
	"Put the status values at the top level, not inside status.",
	`Write the outline as an object with "before" and "after".`,
	"Wrap your answer in a Markdown code block.",
	"Answer in plain prose, without any JSON.",
	`Call the outline field "plot" instead.`,
}

var status = map[string]string{"Health": "good", "Gold": "12"}

// platform is one platform the probe can drive.
type platform struct {
	id     string
	thread func(keys adapters.KeyFunc, model string) adapters.Thread
	prices func(model string) (adapters.Price, bool)
	tiers  []adapters.TierModels
}

// speaking reports a gpt-realtime model, which the Realtime adapter drives.
func speaking(model string) bool { return strings.HasPrefix(model, "gpt-realtime") }

func platforms() []platform {
	return []platform{
		{
			id: "openai",
			thread: func(k adapters.KeyFunc, m string) adapters.Thread {
				if speaking(m) {
					return openai.NewRealtime(k, m, "")
				}
				return openai.NewThread(k, m)
			},
			prices: openai.Prices,
			tiers:  openai.Platform().Tiers,
		},
		{
			id:     "mistral",
			thread: func(k adapters.KeyFunc, m string) adapters.Thread { return mistral.NewThread(k, m) },
			prices: mistral.Prices,
			tiers:  mistral.Platform().Tiers,
		},
	}
}

func main() {
	rounds := flag.Int("rounds", 10, "rounds per model; each makes a fresh call and an appended one")
	only := flag.String("platform", "", "probe only this platform (openai, mistral)")
	flag.Parse()

	schema := genre.OutlineSchema(status)
	// Outline runs on the game master's system message, filled in as a game
	// with these status fields would have it.
	adventure, err := genre.NewAdventure(genre.AdventureConfig{Tool: mock.Tool{}, Thread: mock.Thread{}, Status: status})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	instructions := adventure.Prompts[genre.PromptGameMaster]
	input, _ := json.Marshal(map[string]any{
		"input":  "The player tries to cross the rope bridge in the storm, hoping it holds.",
		"status": status,
	})

	var rows []row
	var spoken []voiceRow
	for _, p := range platforms() {
		if *only != "" && *only != p.id {
			continue
		}
		key, err := devconfig.APIKey(p.id)
		if err != nil || key == "" {
			path, _ := devconfig.Path()
			fmt.Fprintf(os.Stderr, "%s: skipped, no key at platforms.%s.apikey in %s\n", p.id, p.id, path)
			continue
		}
		keys := func(context.Context) (string, error) { return key, nil }
		for _, model := range storyModels(p.tiers) {
			fmt.Fprintf(os.Stderr, "%s %s: %d rounds\n", p.id, model, *rounds)
			start, appended := row{platform: p.id, model: model, call: "start"}, row{platform: p.id, model: model, call: "append"}
			seeded := row{platform: p.id, model: model, call: "seed"}
			thread := p.thread(keys, model)
			for i := range *rounds {
				turn := adapters.ThreadTurn{
					System: instructions + "\n\n" + shapes[i%len(shapes)],
					Input:  string(input),
					Schema: &schema,
				}
				reply := start.probe(thread, turn, schema, p.prices)
				if reply.Continuation.ID == "" {
					appended.skipped++
					seeded.skipped++
					continue
				}
				turn.Continuation = reply.Continuation
				// The appended turn carries the System Message already.
				turn.Transcript = []adapters.Exchange{turn.Exchange("probe", reply.Text)}
				appended.probe(thread, turn, schema, p.prices)

				// A fresh adapter holds no session, as after a restart: the
				// turn opens one seeded with the transcript.
				fresh := p.thread(keys, model)
				seeded.probe(fresh, turn, schema, p.prices)
				closeThread(fresh)
			}
			closeThread(thread)
			rows = append(rows, start, appended, seeded)
			if speaking(model) {
				spoken = append(spoken, speak(p, keys, model, instructions, *rounds))
			}
		}
	}
	report(rows)
	reportVoices(spoken)
}

// closeThread ends what a thread adapter holds open.
func closeThread(thread adapters.Thread) {
	if closer, ok := thread.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

// voiceRow is what a speaking model did on its spoken turns.
type voiceRow struct {
	platform, model string

	spoken, errored int
	// lag is from the transcript's first piece to the first audio chunk;
	// negative when the audio came first.
	lag []time.Duration
	// audioOut is each spoken turn's audio output tokens; audioIn each
	// turn's audio input tokens, which grow if earlier audio stays in the
	// session's context.
	audioOut, audioIn []int64
	cost              float64
	priced            bool
	lastError         string
}

// speak narrates rounds turns in one session, each after the last.
func speak(p platform, keys adapters.KeyFunc, model, instructions string, rounds int) voiceRow {
	r := voiceRow{platform: p.id, model: model}
	thread := p.thread(keys, model)
	defer closeThread(thread)
	fmt.Fprintf(os.Stderr, "%s %s: %d spoken turns\n", p.id, model, rounds)

	var from adapters.Continuation
	var transcript []adapters.Exchange
	for i := range rounds {
		began := time.Now()
		var firstPiece, firstAudio time.Duration
		turn := adapters.ThreadTurn{
			System:       instructions,
			Directive:    true,
			Input:        fmt.Sprintf("NARRATE: The player reaches stretch %d of the rope bridge as the storm grows. 3 sentences.", i+1),
			Speak:        true,
			Continuation: from,
			Transcript:   transcript,
			Piece: func(string) {
				if firstPiece == 0 {
					firstPiece = time.Since(began)
				}
			},
			Audio: func([]byte) {
				if firstAudio == 0 {
					firstAudio = time.Since(began)
				}
			},
		}
		reply, err := thread.Continue(context.Background(), turn)
		if price, ok := p.prices(model); ok {
			r.cost += price.Cost(reply.Usage)
			r.priced = true
		}
		if err != nil {
			r.errored++
			r.lastError = err.Error()
			continue
		}
		r.spoken++
		if firstPiece > 0 && firstAudio > 0 {
			r.lag = append(r.lag, firstAudio-firstPiece)
		}
		r.audioOut = append(r.audioOut, reply.Usage.AudioOutputTokens)
		r.audioIn = append(r.audioIn, reply.Usage.AudioInputTokens+reply.Usage.CachedAudioInputTokens)
		from = reply.Continuation
		transcript = append(transcript, turn.Exchange("probe", reply.Text))
	}
	return r
}

func reportVoices(rows []voiceRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "platform\tmodel\tspoken\terrored\taudio lag\taudio out/turn\taudio in per turn\tcost\t")
	for _, r := range rows {
		cost := "unpriced"
		if r.priced {
			cost = fmt.Sprintf("$%.4f", r.cost)
		}
		in := make([]string, len(r.audioIn))
		for i, n := range r.audioIn {
			in[i] = fmt.Sprint(n)
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t\n",
			r.platform, r.model, r.spoken, r.errored, median(r.lag), medianTokens(r.audioOut), strings.Join(in, " "), cost)
	}
	_ = w.Flush()
	for _, r := range rows {
		if r.lastError != "" {
			fmt.Printf("\n%s %s spoken, last problem: %s", r.platform, r.model, strings.TrimSpace(r.lastError))
		}
	}
	fmt.Println()
}

func medianTokens(ns []int64) string {
	if len(ns) == 0 {
		return "-"
	}
	sorted := slices.Clone(ns)
	slices.Sort(sorted)
	return fmt.Sprint(sorted[len(sorted)/2])
}

// storyModels are a platform's story models, text and speaking, cheapest
// tier first, each once.
func storyModels(tiers []adapters.TierModels) []string {
	var models []string
	for _, kind := range []adapters.ModelType{adapters.ModelStory, adapters.ModelSpeech} {
		for _, t := range tiers {
			if m, ok := t.Models[kind]; ok && !slices.Contains(models, m.Name) {
				models = append(models, m.Name)
			}
		}
	}
	return models
}

// row is what one model did on one kind of call.
type row struct {
	platform, model, call string

	valid, invalid, refused, truncated, errored, skipped int
	// streamed counts replies that arrived in more than one piece.
	streamed int
	first    []time.Duration
	total    []time.Duration
	cost     float64
	priced   bool
	// lastError is the last failure or misfit, to read why.
	lastError string
}

// probe makes one call and counts what came back.
func (r *row) probe(thread adapters.Thread, turn adapters.ThreadTurn, schema adapters.Schema, prices func(string) (adapters.Price, bool)) adapters.ThreadReply {
	began := time.Now()
	pieces := 0
	turn.Piece = func(string) {
		if pieces == 0 {
			r.first = append(r.first, time.Since(began))
		}
		pieces++
	}
	reply, err := thread.Continue(context.Background(), turn)
	r.total = append(r.total, time.Since(began))
	if pieces > 1 {
		r.streamed++
	}
	if price, ok := prices(thread.Model()); ok {
		r.cost += price.Cost(reply.Usage)
		r.priced = true
	}

	switch {
	case errors.Is(err, adapters.ErrRefused):
		r.refused++
	case errors.Is(err, adapters.ErrTruncated):
		r.truncated++
	case err != nil:
		r.errored++
	default:
		if _, err = schema.Check(reply.Text); err != nil {
			r.invalid++
		} else {
			r.valid++
		}
	}
	if err != nil {
		r.lastError = err.Error()
	}
	return reply
}

func report(rows []row) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "platform\tmodel\tcall\tvalid\tinvalid\trefused\ttruncated\terrored\tskipped\tstreamed\tfirst piece\ttotal\tcost\t")
	for _, r := range rows {
		cost := "unpriced"
		if r.priced {
			cost = fmt.Sprintf("$%.4f", r.cost)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t\n",
			r.platform, r.model, r.call, r.valid, r.invalid, r.refused, r.truncated, r.errored, r.skipped,
			r.streamed, median(r.first), median(r.total), cost)
	}
	_ = w.Flush()

	for _, r := range rows {
		if r.lastError != "" {
			fmt.Printf("\n%s %s %s, last problem: %s", r.platform, r.model, r.call, strings.TrimSpace(r.lastError))
		}
	}
	fmt.Println()
}

func median(ds []time.Duration) string {
	if len(ds) == 0 {
		return "-"
	}
	sorted := slices.Clone(ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2].Round(10 * time.Millisecond).String()
}
