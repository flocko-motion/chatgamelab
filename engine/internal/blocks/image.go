package blocks

import (
	"context"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Image turns a prompt into a picture. It is the whole mechanism and nothing
// else: how often it runs is settled by whatever feeds it, so the same block is
// NPC-Live's one portrait when the init stem hands it a single prompt, and
// Adventure's per-turn illustration when the loop does.
type Image struct {
	name  string
	image adapters.Image

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
	return &Image{name: name, image: image, in: make(ports.TextInput, 16)}
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
		defer func() { b.out.Close(); b.state.Close(); b.usage.Close() }()

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
				data, used, err := b.image.Generate(ctx, prompt)
				if err == nil {
					b.report(used)
					b.out.Send(ports.ImageData(data))
				}
				// Reported even when the call failed: a block owes the gate a
				// working-then-ready cycle whatever came of the work, or one
				// broken optional step holds the game shut forever.
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
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
