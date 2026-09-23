package blocks

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"engine/internal/ports"
)

// The output blocks are the sink family: no prompt, no adapter, no model call.
// Which of them a genre wires is what defines that genre's event schema, so
// they are one node per stream rather than one node with four ports.

// recorder is the shared body of every sink: buffer in, readable log out.
type recorder struct {
	name string
	Seen ports.TextInput
}

func newRecorder(name string) recorder {
	return recorder{name: name, Seen: make(ports.TextInput, 256)}
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
	in ports.TextInput
}

func NewPlayerOutputText(name string) *PlayerOutputText {
	return &PlayerOutputText{recorder: newRecorder(name), in: make(ports.TextInput, 64)}
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
	in ports.AudioInput
}

func NewPlayerOutputAudio(name string) *PlayerOutputAudio {
	return &PlayerOutputAudio{recorder: newRecorder(name), in: make(ports.AudioInput, 64)}
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
	in ports.ImageInput
}

func NewPlayerOutputImage(name string) *PlayerOutputImage {
	return &PlayerOutputImage{recorder: newRecorder(name), in: make(ports.ImageInput, 64)}
}

func (b *PlayerOutputImage) ImageInPort() chan<- ports.ImageData { return b.in }
func (b *PlayerOutputImage) RequiredInputs() []ports.Kind        { return []ports.Kind{ports.KindImage} }

func (b *PlayerOutputImage) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			// Encoded here because the session's stream is text: a sink is
			// where a picture stops being bytes and becomes something a page
			// can show.
			if !b.record(ctx, dataURL(v)) {
				return
			}
		}
	}()
}

// dataURL wraps image bytes so a client can display them without a second
// request. A placeholder adapter that returns text rather than an image is
// passed through unchanged, so a stub stays readable.
func dataURL(data ports.ImageData) string {
	if !bytes.HasPrefix(data, []byte("\x89PNG")) {
		return string(data)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

type PlayerOutputProps struct {
	recorder
	in ports.PropsInput
}

func NewPlayerOutputProps(name string) *PlayerOutputProps {
	return &PlayerOutputProps{recorder: newRecorder(name), in: make(ports.PropsInput, 64)}
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
