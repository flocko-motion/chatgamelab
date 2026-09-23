package blocks

import (
	"context"
	"fmt"

	"engine/internal/ports"
)

// The output blocks are the sink family: no prompt, no adapter, no model call.
// Which of them a genre wires is what defines that genre's event schema, so
// they are one node per stream rather than one node with four ports.

// recorder is the shared body of every sink: buffer in, readable log out.
type recorder struct {
	name string
	Seen chan string
}

func newRecorder(name string) recorder {
	return recorder{name: name, Seen: make(chan string, 256)}
}

func (r *recorder) NodeName() string { return r.name }

func (r *recorder) record(ctx context.Context, v string) bool {
	select {
	case r.Seen <- v:
		return true
	case <-ctx.Done():
		return false
	}
}

type PlayerOutputText struct {
	recorder
	in chan string
}

func NewPlayerOutputText(name string) *PlayerOutputText {
	return &PlayerOutputText{recorder: newRecorder(name), in: make(chan string, 64)}
}

func (b *PlayerOutputText) TextInPort() chan<- string    { return b.in }
func (b *PlayerOutputText) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

func (b *PlayerOutputText) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			if !b.record(ctx, v) {
				return
			}
		}
	}()
}

type PlayerOutputAudio struct {
	recorder
	in chan ports.AudioChunk
}

func NewPlayerOutputAudio(name string) *PlayerOutputAudio {
	return &PlayerOutputAudio{recorder: newRecorder(name), in: make(chan ports.AudioChunk, 64)}
}

func (b *PlayerOutputAudio) AudioInPort() chan<- ports.AudioChunk { return b.in }
func (b *PlayerOutputAudio) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindAudio} }

func (b *PlayerOutputAudio) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			if !b.record(ctx, string(v)) {
				return
			}
		}
	}()
}

type PlayerOutputImage struct {
	recorder
	in chan ports.ImageData
}

func NewPlayerOutputImage(name string) *PlayerOutputImage {
	return &PlayerOutputImage{recorder: newRecorder(name), in: make(chan ports.ImageData, 64)}
}

func (b *PlayerOutputImage) ImageInPort() chan<- ports.ImageData { return b.in }
func (b *PlayerOutputImage) RequiredInputs() []ports.Kind        { return []ports.Kind{ports.KindImage} }

func (b *PlayerOutputImage) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			if !b.record(ctx, string(v)) {
				return
			}
		}
	}()
}

type PlayerOutputProps struct {
	recorder
	in chan ports.PropMap
}

func NewPlayerOutputProps(name string) *PlayerOutputProps {
	return &PlayerOutputProps{recorder: newRecorder(name), in: make(chan ports.PropMap, 64)}
}

func (b *PlayerOutputProps) PropsInPort() chan<- ports.PropMap { return b.in }
func (b *PlayerOutputProps) RequiredInputs() []ports.Kind      { return []ports.Kind{ports.KindProps} }

func (b *PlayerOutputProps) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			if !b.record(ctx, fmt.Sprint(map[string]string(v))) {
				return
			}
		}
	}()
}
