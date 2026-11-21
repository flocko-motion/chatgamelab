package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/devconfig"
	"engine/internal/genre"
)

// clip is one clip the hearing models hear, and the words it holds; none for
// silence.
type clip struct {
	words string
	pcm   adapters.Clip
}

// hearingRow is what one hearing model did.
type hearingRow struct {
	platform, model string

	valid, invalid, refused, truncated, errored int
	// faithful is, per spoken clip heard, the share of its words "heard"
	// holds; silent counts silence heard as nothing, of silences heard.
	faithful        []float64
	silent, silence int
	latency         []time.Duration
	seconds, cost   float64
	priced          bool
	lastError       string
	// heard keeps one answer per clip, by its place, to read; misheard every
	// answer that missed words.
	heard    map[int]string
	misheard []string
}

func probeHearing(only, dir, language string, rounds int) {
	clips, err := readClips(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	adventure, err := genre.NewAdventure(genre.AdventureConfig{Tool: mock.Tool{}, Hearing: mock.Hearing{}, Thread: mock.Thread{}, Image: mock.Image{}, Language: language})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	system, schema := adventure.Prompts[genre.PromptRephraseAudio], genre.SpokenSchema()

	var rows []hearingRow
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
		for _, model := range slices.Sorted(func(yield func(string) bool) {
			for m := range p.hearing {
				if !yield(m) {
					return
				}
			}
		}) {
			fmt.Fprintf(os.Stderr, "%s %s: %d clips, %d rounds\n", p.id, model, len(clips), rounds)
			r := hearingRow{platform: p.id, model: model, heard: map[int]string{}}
			hear := p.hearing[model](keys)
			for range rounds {
				for i, c := range clips {
					r.probe(hear, system, schema, c, p.prices, i)
				}
			}
			rows = append(rows, r)
		}
	}
	reportHearing(rows, clips)
}

func (r *hearingRow) probe(hear adapters.Hearing, system string, schema adapters.Schema, c clip, prices adapters.ModelTable, i int) {
	began := time.Now()
	reply, used, err := hear.Hear(context.Background(), system, c.pcm, schema)
	r.latency = append(r.latency, time.Since(began))
	r.seconds += c.pcm.Seconds()
	if price, ok := prices.Price(used.Model); ok {
		r.cost += price.Cost(used)
		r.priced = true
	}
	switch {
	case errors.Is(err, adapters.ErrRefused):
		r.refused++
	case errors.Is(err, adapters.ErrTruncated):
		r.truncated++
	case err != nil:
		r.errored++
	}
	if err != nil {
		r.lastError = err.Error()
		return
	}
	fields, err := schema.Check(adapters.JSONObject(reply))
	if err != nil {
		r.invalid++
		r.lastError = err.Error() + ": " + reply
		return
	}
	r.valid++
	heard, _ := fields["heard"].(string)
	action, _ := fields["action"].(string)
	if _, kept := r.heard[i]; !kept {
		r.heard[i] = fmt.Sprintf("%q → %q", heard, action)
	}
	if c.words == "" {
		r.silence++
		if strings.TrimSpace(heard) == "" {
			r.silent++
		}
		return
	}
	share := overlap(c.words, heard)
	r.faithful = append(r.faithful, share)
	if share < 1 {
		r.misheard = append(r.misheard, fmt.Sprintf("%s: %q", c.words, heard))
	}
}

// overlap is the share of want's words that got holds, ignoring case and
// punctuation.
func overlap(want, got string) float64 {
	have := map[string]int{}
	for _, w := range words(got) {
		have[w]++
	}
	wanted := words(want)
	found := 0
	for _, w := range wanted {
		if have[w] > 0 {
			have[w]--
			found++
		}
	}
	return float64(found) / float64(max(len(wanted), 1))
}

// words are a text's words, lower case, with accents folded away: a file
// name on macOS spells "ü" as "u" and a combining mark, a model as one letter.
func words(s string) []string {
	var folded strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		folded.WriteRune(fold(r))
	}
	return strings.FieldsFunc(folded.String(), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}

// accented are the accented letters fold reads as their base letter.
var accented = map[rune]string{
	'a': "àáâãäåā", 'c': "çćč", 'e': "èéêëēė", 'i': "ìíîïī", 'n': "ñń",
	'o': "òóôõöøō", 's': "śš", 'u': "ùúûüū", 'y': "ýÿ", 'z': "źżž",
}

func fold(r rune) rune {
	for base, letters := range accented {
		if strings.ContainsRune(letters, r) {
			return base
		}
	}
	return r
}

// readClips reads every WAV in dir, named after its words, and adds two
// seconds of silence.
func readClips(dir string) ([]clip, error) {
	var clips []clip
	if dir != "" {
		paths, err := filepath.Glob(filepath.Join(dir, "*.wav"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			pcm, err := readWAV(path)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			clips = append(clips, clip{words: strings.TrimSuffix(filepath.Base(path), ".wav"), pcm: pcm})
		}
	}
	if len(clips) == 0 {
		fmt.Fprintln(os.Stderr, "no clips: give -clips a directory of WAV files; only silence is heard")
	}
	return append(clips, clip{pcm: mock.SilentClip(2)}), nil
}

// readWAV reads a WAV file's samples, which must be PCM16 at 24 kHz mono.
func readWAV(path string) (adapters.Clip, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, errors.New("not a WAV file")
	}
	for at := 12; at+8 <= len(data); {
		id, size := string(data[at:at+4]), int(binary.LittleEndian.Uint32(data[at+4:]))
		body := data[at+8 : min(at+8+size, len(data))]
		switch id {
		case "fmt ":
			if len(body) < 16 {
				return nil, errors.New("a short fmt chunk")
			}
			format, channels := binary.LittleEndian.Uint16(body), binary.LittleEndian.Uint16(body[2:])
			rate, bits := binary.LittleEndian.Uint32(body[4:]), binary.LittleEndian.Uint16(body[14:])
			if format != 1 || channels != 1 || rate != adapters.ClipRate || bits != 16 {
				return nil, fmt.Errorf("format %d, %d channels, %d Hz, %d bits: want PCM16 at 24 kHz mono", format, channels, rate, bits)
			}
		case "data":
			return adapters.Clip(body), nil
		}
		at += 8 + size + size%2
	}
	return nil, errors.New("no data chunk")
}

func reportHearing(rows []hearingRow, clips []clip) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "platform\tmodel\tvalid\tinvalid\trefused\ttruncated\terrored\tfaithful\tsilence\tlatency\tper spoken minute\t")
	for _, r := range rows {
		cost := "unpriced"
		if r.priced && r.seconds > 0 {
			cost = fmt.Sprintf("$%.4f", r.cost/r.seconds*60)
		}
		faithful := "-"
		if len(r.faithful) > 0 {
			sum := 0.0
			for _, f := range r.faithful {
				sum += f
			}
			faithful = fmt.Sprintf("%.0f%%", sum/float64(len(r.faithful))*100)
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%d/%d\t%s\t%s\t\n",
			r.platform, r.model, r.valid, r.invalid, r.refused, r.truncated, r.errored, faithful, r.silent, r.silence, median(r.latency), cost)
	}
	_ = w.Flush()
	for _, r := range rows {
		fmt.Printf("\n%s %s heard:\n", r.platform, r.model)
		for i, c := range clips {
			words := c.words
			if words == "" {
				words = "(silence)"
			}
			fmt.Printf("  %s: %s\n", words, r.heard[i])
		}
		for _, m := range r.misheard {
			fmt.Printf("  misheard %s\n", m)
		}
		if r.lastError != "" {
			fmt.Printf("  last problem: %s\n", strings.TrimSpace(r.lastError))
		}
	}
}
