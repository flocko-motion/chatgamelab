package blocks

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"engine/internal/ports"
)

// DummyToolCall stands in for a single-shot text transformation. The role is
// config, not type: the same block is a rephrase, a translation or a summary
// depending on what it was constructed with. A real AiToolCall replaces the
// body and keeps the ports.
type DummyToolCall struct {
	name, role string
	in         chan string
	out        ports.TextBroadcast
}

func NewDummyToolCall(name, role string) *DummyToolCall {
	return &DummyToolCall{name: name, role: role, in: make(chan string, 16)}
}

func (b *DummyToolCall) NodeName() string           { return b.name }
func (b *DummyToolCall) TextInPort() chan<- string  { return b.in }
func (b *DummyToolCall) TextOutPort() <-chan string { return b.out.Subscribe() }
func (b *DummyToolCall) RequiredInputs() []ports.Kind {
	return []ports.Kind{ports.KindText}
}

func (b *DummyToolCall) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.out.Send(fmt.Sprintf("[%s] %s", b.role, v))
			}
		}
	}()
}

// DummyThreaded stands in for a call on a continuing conversation. It reports
// its turn number so a wiring test can see the thread is actually continuous.
type DummyThreaded struct {
	name, role string
	turn       int
	// thread stands in for the provider's continuation token — the thing that
	// makes this role threaded, and the thing a resumed session needs back.
	thread string
	in     chan string
	out    ports.TextBroadcast
}

func (b *DummyThreaded) ExportState() string { return b.thread }

func (b *DummyThreaded) RestoreState(s string) {
	b.thread = s
	// A restored thread continues its numbering rather than restarting, which
	// is how a test can tell a real resume from a fresh session.
	if n, err := strconv.Atoi(strings.TrimPrefix(s, "resp-")); err == nil {
		b.turn = n
	}
}

func NewDummyThreaded(name, role string) *DummyThreaded {
	return &DummyThreaded{name: name, role: role, in: make(chan string, 16)}
}

func (b *DummyThreaded) NodeName() string             { return b.name }
func (b *DummyThreaded) TextInPort() chan<- string    { return b.in }
func (b *DummyThreaded) TextOutPort() <-chan string   { return b.out.Subscribe() }
func (b *DummyThreaded) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

func (b *DummyThreaded) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.turn++
				b.thread = fmt.Sprintf("resp-%d", b.turn)
				b.out.Send(fmt.Sprintf("[%s turn %d] %s", b.role, b.turn, v))
			}
		}
	}()
}

// DummyExtraction is the block with two text outputs, which is the case that
// forced SecondaryTextOut: Outline emits a plot line and an image prompt, both
// strings, and they must not be interchangeable at a call site.
type DummyExtraction struct {
	name      string
	in        chan string
	propsIn   chan ports.PropMap
	out       ports.TextBroadcast
	secondary ports.TextBroadcast
	props     ports.PropsBroadcast

	mu      sync.Mutex
	current ports.PropMap
	turn    int
}

func NewDummyExtraction(name string) *DummyExtraction {
	return &DummyExtraction{
		name:    name,
		in:      make(chan string, 16),
		propsIn: make(chan ports.PropMap, 16),
		current: ports.PropMap{},
	}
}

func (b *DummyExtraction) NodeName() string                    { return b.name }
func (b *DummyExtraction) TextInPort() chan<- string           { return b.in }
func (b *DummyExtraction) PropsInPort() chan<- ports.PropMap   { return b.propsIn }
func (b *DummyExtraction) TextOutPort() <-chan string          { return b.out.Subscribe() }
func (b *DummyExtraction) SecondaryTextOutPort() <-chan string { return b.secondary.Subscribe() }
func (b *DummyExtraction) PropsOutPort() <-chan ports.PropMap  { return b.props.Subscribe() }

// Props are required rather than optional: the current values arrive on an edge,
// so a genre that forgets to wire them fails validation instead of silently
// relying on the model's memory.
func (b *DummyExtraction) RequiredInputs() []ports.Kind {
	return []ports.Kind{ports.KindText, ports.KindProps}
}

func (b *DummyExtraction) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.secondary.Close(); b.props.Close() }()
		for {
			select {
			case <-ctx.Done():
				return

			case update, ok := <-b.propsIn:
				if !ok {
					return
				}
				b.mu.Lock()
				for k, v := range update {
					b.current[k] = v
				}
				b.mu.Unlock()

			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.mu.Lock()
				b.turn++
				turn := b.turn
				// A real block would hand the current values to the model and
				// return what it decided; the stub just carries them forward so
				// the edge is observably doing something.
				health := b.current["Health"]
				if health == "" {
					health = "Good"
				}
				b.mu.Unlock()

				b.out.Send(fmt.Sprintf("[outline] world reacts to: %s", v))
				b.secondary.Send("[image-prompt] dim stone bridge, torchlight")
				b.props.Send(ports.PropMap{"Health": health, "Turn": fmt.Sprint(turn)})
			}
		}
	}()
}

type DummyImage struct {
	name string
	in   chan string
	out  ports.ImageBroadcast
}

func NewDummyImage(name string) *DummyImage {
	return &DummyImage{name: name, in: make(chan string, 16)}
}

func (b *DummyImage) NodeName() string                     { return b.name }
func (b *DummyImage) TextInPort() chan<- string            { return b.in }
func (b *DummyImage) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }
func (b *DummyImage) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindText} }

func (b *DummyImage) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.out.Send(ports.ImageData("<png of " + strings.TrimPrefix(v, "[image-prompt] ") + ">"))
			}
		}
	}()
}

type DummyTTS struct {
	name string
	in   chan string
	out  ports.AudioBroadcast
}

func NewDummyTTS(name string) *DummyTTS { return &DummyTTS{name: name, in: make(chan string, 16)} }

func (b *DummyTTS) NodeName() string                      { return b.name }
func (b *DummyTTS) TextInPort() chan<- string             { return b.in }
func (b *DummyTTS) AudioOutPort() <-chan ports.AudioChunk { return b.out.Subscribe() }
func (b *DummyTTS) RequiredInputs() []ports.Kind          { return []ports.Kind{ports.KindText} }

func (b *DummyTTS) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.out.Send(ports.AudioChunk("<narration of " + v + ">"))
			}
		}
	}()
}
