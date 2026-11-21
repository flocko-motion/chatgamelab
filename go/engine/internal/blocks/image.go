package blocks

import (
	"context"
	"errors"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Image turns a prompt into a picture. It is the whole mechanism and nothing
// else: how often it runs is settled by whatever feeds it, so the same block is
// NPC-Live's one portrait when the init stem hands it a single prompt, and
// Adventure's per-turn illustration when the loop does.
//
// A failure that may pass is retried after each Backoff; a call that fails
// for good sends its prompt on the failure output.
type Image struct {
	Switch
	failing
	name  string
	image adapters.Image

	// Backoff is the wait before each retry of a failure that may pass. Set
	// before Start.
	Backoff []time.Duration

	in    ports.TextInput
	out   ports.ImageBroadcast
	state ports.StateBroadcast
	usage ports.UsageBroadcast

	spent adapters.Usage
}

func NewImage(name string, image adapters.Image) *Image {
	// A missing adapter is a wiring mistake, and it should surface where the
	// graph is built rather than as a nil dereference inside a goroutine three
	// turns into someone's conversation.
	if image == nil {
		panic("blocks: " + name + " needs an image adapter")
	}
	return &Image{
		name:    name,
		image:   image,
		Backoff: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
		in:      make(ports.TextInput, 16),
	}
}

func (b *Image) NodeName() string                     { return b.name }
func (b *Image) TextInPort() chan<- string            { return b.in }
func (b *Image) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }
func (b *Image) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindText} }

// UsageOutPort reports what the pictures have cost.
func (b *Image) UsageOutPort() <-chan ports.Usage { return b.usage.Subscribe() }

// StateOutPort reports what this block is doing. Wire it to the gate only if the
// game should wait for the picture; a portrait usually should not.
func (b *Image) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *Image) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.failure.Close(); b.state.Close(); b.usage.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		for {
			select {
			case <-ctx.Done():
				return
			case prompt, open := <-b.in:
				if !open {
					return
				}

				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				// Off, the prompt is dropped; the cycle still completes for the gate.
				if !b.Active() {
					b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
					continue
				}
				data, err := b.generate(ctx, prompt)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					// Ready all the same: a block owes the gate a
					// working-then-ready cycle whatever came of the work.
					b.fail(&b.state, b.name, prompt, err)
					continue
				}
				b.out.Send(ports.ImageData(data))
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}

// generate makes the picture, retrying a failure that may pass after each backoff.
func (b *Image) generate(ctx context.Context, prompt string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		data, used, err := b.image.Generate(ctx, adapters.ImageRequest{Prompt: prompt})
		if err == nil {
			b.report(used)
			return data, nil
		}
		if !errors.Is(err, adapters.ErrTransient) {
			return nil, err
		}
		if attempt == len(b.Backoff) {
			return nil, errExhausted{err}
		}
		if !sleep(ctx, b.Backoff[attempt]) {
			return nil, ctx.Err()
		}
	}
}

// report accumulates and publishes the running total, so a late subscriber sees
// everything spent rather than only what came after it.
func (b *Image) report(used adapters.Usage) {
	b.spent.Add(used)
	b.usage.Send(ports.Usage{
		Node:         b.name,
		Model:        b.spent.Model,
		InputTokens:  b.spent.InputTokens,
		OutputTokens: b.spent.OutputTokens,
		Images:       b.spent.Images,
	})
}
