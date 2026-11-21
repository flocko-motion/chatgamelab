package blocks

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"

	"engine/internal/ports"
)

// The output blocks are the sink family: no prompt, no adapter, no model call.
// Which of them a genre wires is what defines that genre's event schema, so
// they are one node per stream rather than one node with four ports.

// recorder is the shared body of every sink: buffer in, readable log out.
type recorder struct {
	name   string
	stream string
	seen   ports.TextInput
}

func newRecorder(name, stream string) recorder {
	return recorder{name: name, stream: stream, seen: make(ports.TextInput, 256)}
}

func (r *recorder) NodeName() string { return r.name }

// Stream names this sink on the wire, and is set where the node is built so the
// two cannot be given different answers.
func (r *recorder) Stream() string { return r.stream }

func (r *recorder) Arrivals() chan string { return r.seen }

func (r *recorder) record(ctx context.Context, v string) bool {
	select {
	case r.seen <- v:
		return true
	case <-ctx.Done():
		return false
	}
}

type PlayerOutputText struct {
	recorder
	in ports.TextInput
}

func NewPlayerOutputText(name, stream string) *PlayerOutputText {
	return &PlayerOutputText{recorder: newRecorder(name, stream), in: make(ports.TextInput, 64)}
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

func NewPlayerOutputAudio(name, stream string) *PlayerOutputAudio {
	return &PlayerOutputAudio{recorder: newRecorder(name, stream), in: make(ports.AudioInput, 64)}
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

func NewPlayerOutputImage(name, stream string) *PlayerOutputImage {
	return &PlayerOutputImage{recorder: newRecorder(name, stream), in: make(ports.ImageInput, 64)}
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

func NewPlayerOutputProps(name, stream string) *PlayerOutputProps {
	return &PlayerOutputProps{recorder: newRecorder(name, stream), in: make(ports.PropsInput, 64)}
}

func (b *PlayerOutputProps) PropsInPort() chan<- ports.PropMap { return b.in }
func (b *PlayerOutputProps) RequiredInputs() []ports.Kind      { return []ports.Kind{ports.KindProps} }

func (b *PlayerOutputProps) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			// JSON, so a value with spaces or colons arrives whole.
			blob, err := json.Marshal(map[string]string(v))
			if err != nil {
				continue
			}
			if !b.record(ctx, string(blob)) {
				return
			}
		}
	}()
}

// PlayerOutputStream is the text sink for a reply streamed piece by piece: each
// piece goes out as it arrives, with the close and the discard as they come,
// so the player watches the prose appear.
type PlayerOutputStream struct {
	recorder
	in ports.StreamInput
}

func NewPlayerOutputStream(name, stream string) *PlayerOutputStream {
	return &PlayerOutputStream{recorder: newRecorder(name, stream), in: make(ports.StreamInput, 256)}
}

func (b *PlayerOutputStream) StreamInPort() chan<- ports.Piece { return b.in }
func (b *PlayerOutputStream) RequiredInputs() []ports.Kind     { return []ports.Kind{ports.KindStream} }

func (b *PlayerOutputStream) Start(ctx context.Context) {
	go func() {
		for v := range b.in {
			if !b.record(ctx, string(v)) {
				return
			}
		}
	}()
}

// ErrorSink is where a failure the game goes on after ends: each goes out on
// the `error` stream with the sink's code and the failed call's input, as the
// JSON object {"code", "text"}, so the playback-engine tells the player in
// their language, and may give the text back.
type ErrorSink struct {
	recorder
	code string
	in   ports.FailureInput
}

func NewErrorSink(name, code string) *ErrorSink {
	return &ErrorSink{recorder: newRecorder(name, "error"), code: code, in: make(ports.FailureInput, 16)}
}

func (b *ErrorSink) FailureInPort() chan<- ports.Failure { return b.in }
func (b *ErrorSink) RequiredInputs() []ports.Kind        { return []ports.Kind{ports.KindFailure} }

func (b *ErrorSink) Start(ctx context.Context) {
	go func() {
		for f := range b.in {
			blob, err := json.Marshal(map[string]string{"code": b.code, "text": f.Input})
			if err != nil {
				continue
			}
			if !b.record(ctx, string(blob)) {
				return
			}
		}
	}()
}

// ProblemSink is where a failure the game cannot go on after ends: each
// arrives as the JSON object {"node", "reason", "message"}, without the input,
// for the session to put on the `problem` stream, named by its reason, and end.
type ProblemSink struct {
	recorder
	in ports.FailureInput
}

func NewProblemSink(name string) *ProblemSink {
	return &ProblemSink{recorder: newRecorder(name, "problem"), in: make(ports.FailureInput, 16)}
}

func (b *ProblemSink) FailureInPort() chan<- ports.Failure { return b.in }
func (b *ProblemSink) RequiredInputs() []ports.Kind        { return []ports.Kind{ports.KindFailure} }

func (b *ProblemSink) Start(ctx context.Context) {
	go func() {
		for f := range b.in {
			f.Input = ""
			blob, err := json.Marshal(f)
			if err != nil {
				continue
			}
			if !b.record(ctx, string(blob)) {
				return
			}
		}
	}()
}
