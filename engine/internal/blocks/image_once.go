package blocks

import (
	"context"
	"sync"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// ImageOnce generates a single picture for the whole session and then never
// again. NPC-Live needs exactly this: one portrait of the character the player
// is talking to, made at the start, while the conversation itself is ephemeral.
//
// It is stateful on purpose — "have I already done this" is the whole block —
// which also makes it the one thing a live session has worth persisting.
type ImageOnce struct {
	name  string
	image adapters.Image

	in  chan string
	out ports.ImageBroadcast

	mu   sync.Mutex
	done bool
}

func NewImageOnce(name string, image adapters.Image) *ImageOnce {
	// A missing adapter is a wiring mistake, and it should surface where the
	// graph is built rather than as a nil dereference inside a goroutine three
	// turns into someone's conversation.
	if image == nil {
		panic("blocks: " + name + " needs an image adapter")
	}
	return &ImageOnce{name: name, image: image, in: make(chan string, 4)}
}

func (b *ImageOnce) NodeName() string                     { return b.name }
func (b *ImageOnce) TextInPort() chan<- string            { return b.in }
func (b *ImageOnce) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }
func (b *ImageOnce) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindText} }

// ExportState records that the picture exists, not the picture itself. A
// resumed session re-serves the stored image rather than paying to make a
// second one — which would also be a different picture.
func (b *ImageOnce) ExportState() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.done {
		return "generated"
	}
	return ""
}

func (b *ImageOnce) RestoreState(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.done = s == "generated"
}

// claim reports whether this call is the one that gets to generate.
func (b *ImageOnce) claim() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.done {
		return false
	}
	b.done = true
	return true
}

func (b *ImageOnce) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case prompt, open := <-b.in:
				if !open {
					return
				}
				if !b.claim() {
					continue
				}
				data, err := b.image.Generate(ctx, prompt)
				if err != nil {
					// A missing portrait is not worth ending a conversation
					// over, so the failure is reported and the session runs on.
					b.mu.Lock()
					b.done = false
					b.mu.Unlock()
					continue
				}
				b.out.Send(ports.ImageData(data))
			}
		}
	}()
}
