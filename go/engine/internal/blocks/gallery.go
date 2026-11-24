package blocks

import (
	"context"

	"engine/internal/ports"
)

// GalleryImage shows a picture that already exists, such as the game's cover
// from the gallery, instead of painting one. It takes the place of Image where
// a genre wants to start at once and spend nothing: the same image output, and
// the same working-then-ready cycle for a gate. It shows its picture once per
// session, at startup.
type GalleryImage struct {
	name    string
	picture []byte

	out   ports.ImageBroadcast
	state ports.StateBroadcast

	// journal says when there was no picture to show.
	journal *ports.Journal

	// once remembers the picture was shown, so a resumed session does not
	// show it twice: the history already holds it.
	once
}

// NewGalleryImage builds the block around its picture; nil shows nothing,
// and the cycle still completes.
func NewGalleryImage(name string, picture []byte) *GalleryImage {
	return &GalleryImage{name: name, picture: picture}
}

func (b *GalleryImage) NodeName() string                     { return b.name }
func (b *GalleryImage) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }
func (b *GalleryImage) StateOutPort() <-chan ports.State     { return b.state.Subscribe() }
func (b *GalleryImage) UseJournal(j *ports.Journal)          { b.journal = j }

func (b *GalleryImage) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		already := b.done()
		// Shown before: owed to a gate all the same, as done, with nothing to show.
		if already {
			b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady, Done: true})
			return
		}
		// Owed to a gate even when there is nothing to show, or the game never starts.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})

		switch {
		case len(b.picture) == 0:
			b.journal.Warn(b.name, "no-picture", "the game has no picture in the gallery")
		case ctx.Err() == nil:
			b.out.Send(ports.ImageData(b.picture))
			b.mark()
		}

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}()
}
