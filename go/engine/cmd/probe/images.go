package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // the size of a JPEG picture
	_ "image/png"  // the size of a PNG picture
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/openai"
	"engine/internal/devconfig"
)

// probeImagePrompt is the scene every image probe draws: a game's kind of
// picture, so latency and cost are what a session would see.
const probeImagePrompt = "A lighthouse on a rocky coast at dusk, waves breaking, a small boat in the distance, painted in a storybook style."

// imageSetting is one picture setting a tier draws with: its model and the
// parameters it is called with, and the tiers that use it.
type imageSetting struct {
	model, quality, size string
	tiers                []string
}

// imageRow is what the rounds of one setting brought.
type imageRow struct {
	platform string
	setting  imageSetting
	drawn    int
	errored  int
	latency  []time.Duration
	cost     float64
	priced   bool
	// got is the size of the last picture drawn, as WIDTHxHEIGHT, and its bytes.
	got       string
	bytes     int
	lastError string
}

// imagePlatform is one platform the probe draws pictures on.
type imagePlatform struct {
	id     string
	image  func(keys adapters.KeyFunc, model, quality, size string) adapters.Image
	prices adapters.ModelTable
	tiers  []adapters.TierModels
}

func imagePlatforms() []imagePlatform {
	return []imagePlatform{{
		id: "openai",
		image: func(k adapters.KeyFunc, model, quality, size string) adapters.Image {
			return openai.NewImage(k, model, quality, size)
		},
		prices: openai.Models,
		tiers:  openai.Platform().Tiers,
	}}
}

// probeImages draws pictures with every setting a platform's tiers use, as a
// session's image block would: the tier's model, quality and size. It counts
// the pictures drawn and the failures, the latency, the cost per picture and
// the size the picture came back in. With out, it keeps the pictures there to
// look at.
func probeImages(only, out string, rounds int) {
	var rows []imageRow
	for _, p := range imagePlatforms() {
		if only != "" && only != p.id {
			continue
		}
		key, err := devconfig.APIKey(p.id)
		if err != nil || key == "" {
			fmt.Fprintf(os.Stderr, "%s: skipped, no key\n", p.id)
			continue
		}
		keys := func(context.Context) (string, error) { return key, nil }
		for _, s := range imageSettings(p.tiers) {
			fmt.Fprintf(os.Stderr, "%s %s %s %s (%s): %d rounds\n", p.id, s.model, s.quality, s.size, strings.Join(s.tiers, ", "), rounds)
			r := imageRow{platform: p.id, setting: s}
			draw := p.image(keys, s.model, s.quality, s.size)
			for i := range rounds {
				picture := r.probe(draw, p.prices)
				if out != "" && picture != nil {
					name := fmt.Sprintf("%s-%s-%s-%s-%d.img", p.id, s.model, s.quality, s.size, i+1)
					if err := os.WriteFile(filepath.Join(out, name), picture, 0o644); err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				}
			}
			rows = append(rows, r)
		}
	}
	reportImages(rows)
}

// imageSettings are the distinct picture settings of the tiers, cheapest tier
// first, each with the tiers that draw with it.
func imageSettings(tiers []adapters.TierModels) []imageSetting {
	var out []imageSetting
	for _, tier := range tiers {
		m, ok := tier.Models[adapters.ModelImage]
		if !ok {
			continue
		}
		s := imageSetting{model: m.Name, quality: m.Params[adapters.ParamQuality], size: m.Params[adapters.ParamSize]}
		found := false
		for i := range out {
			if out[i].model == s.model && out[i].quality == s.quality && out[i].size == s.size {
				out[i].tiers = append(out[i].tiers, string(tier.Tier))
				found = true
			}
		}
		if !found {
			s.tiers = []string{string(tier.Tier)}
			out = append(out, s)
		}
	}
	return out
}

// probe draws one picture and records it; it answers the picture, nil when
// none came.
func (r *imageRow) probe(draw adapters.Image, prices adapters.ModelTable) []byte {
	began := time.Now()
	picture, used, err := draw.Generate(context.Background(), adapters.ImageRequest{Prompt: probeImagePrompt})
	r.latency = append(r.latency, time.Since(began))
	if price, ok := prices.Price(used.Model); ok {
		r.cost += price.Cost(used)
		r.priced = true
	}
	if err != nil {
		r.errored++
		r.lastError = err.Error()
		return nil
	}
	r.drawn++
	r.bytes = len(picture)
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(picture)); err == nil {
		r.got = fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)
	} else {
		r.got = "?"
	}
	return picture
}

func reportImages(rows []imageRow) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "platform\tmodel\tquality\tsize\ttiers\tdrawn\terrored\tlatency\tper picture\tgot\tbytes\t")
	for _, r := range rows {
		cost := "unpriced"
		if r.priced && r.drawn+r.errored > 0 {
			cost = fmt.Sprintf("$%.4f", r.cost/float64(r.drawn+r.errored))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%d\t\n",
			r.platform, r.setting.model, r.setting.quality, r.setting.size, strings.Join(r.setting.tiers, ","),
			r.drawn, r.errored, median(r.latency), cost, r.got, r.bytes)
	}
	_ = w.Flush()
	for _, r := range rows {
		if r.lastError != "" {
			fmt.Printf("\n%s %s last problem: %s\n", r.platform, r.setting.model, strings.TrimSpace(r.lastError))
		}
	}
}
