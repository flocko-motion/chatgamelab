package blocks

import (
	"context"
	"sync"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// ImageOnce generates a single picture for the whole session, during the
// preparation phase, and then never again. NPC-Live needs exactly this: one
// portrait of the character, made before the conversation starts, while the
// conversation itself stays ephemeral.
//
// It takes no input edge. Its prompt is configuration, and the work happens in
// Prepare rather than in response to something flowing through the graph — which
// is what makes it an init block rather than a stage.
type ImageOnce struct {
	name   string
	image  adapters.Image
	prompt string

	out   ports.ImageBroadcast
	state ports.StateBroadcast

	mu        sync.Mutex
	data      ports.ImageData
	generated bool
}

func NewImageOnce(name string, image adapters.Image, prompt string) *ImageOnce {
	// A missing adapter is a wiring mistake, and it should surface where the
	// graph is built rather than as a nil dereference inside a goroutine three
	// turns into someone's conversation.
	if image == nil {
		panic("blocks: " + name + " needs an image adapter")
	}
	return &ImageOnce{name: name, image: image, prompt: prompt}
}

func (b *ImageOnce) NodeName() string                     { return b.name }
func (b *ImageOnce) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }

// StateOutPort reports what this block is doing. Wire it to the gate only if the
// game should wait for the picture; a portrait usually should not.
func (b *ImageOnce) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// ExportState records that the picture exists, not the picture itself. The
// image is persisted as the session's one durable artifact and re-served from
// there.
func (b *ImageOnce) ExportState() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.generated {
		return "generated"
	}
	return ""
}

func (b *ImageOnce) RestoreState(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.generated = s == "generated"
}

// Start does the one-time work and then reports done. Nothing triggers it: the
// prompt is configuration, which is what makes this an init block rather than a
// stage in the loop.
func (b *ImageOnce) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		b.mu.Lock()
		already := b.generated
		b.mu.Unlock()

		// Reported even when there is nothing to generate: a gating block owes
		// the gate a working-then-ready cycle, or the game never starts.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})

		// A resumed session skips generation: the image already exists in
		// storage, and paying again would produce a different picture. The
		// client fetches the stored one, and the gate is still told we are done.
		if !already {
			data, err := b.image.Generate(ctx, b.prompt)
			if err == nil {
				b.mu.Lock()
				b.data = ports.ImageData(data)
				b.generated = true
				b.mu.Unlock()
			}
		}

		b.mu.Lock()
		data, has := b.data, len(b.data) > 0
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		default:
		}
		if has {
			b.out.Send(data)
		}
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}()
}
