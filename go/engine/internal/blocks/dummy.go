package blocks

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// The dummies bill invented numbers, priced from the mock's table, so the usage
// view means something without a real model behind it.

// DummyToolCall stands in for a single-shot text transformation. The role is
// config, not type: the same block is a rephrase, a translation or a summary
// depending on what it was constructed with. A real AiToolCall replaces the
// body and keeps the ports.
type DummyToolCall struct {
	name, role string
	in         ports.TextInput
	out        ports.TextBroadcast
	meter      meter
}

func NewDummyToolCall(name, role string) *DummyToolCall {
	return &DummyToolCall{name: name, role: role, in: make(ports.TextInput, 16), meter: meter{name: name}}
}

// UsageOutPort reports what the calls have cost.
func (b *DummyToolCall) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

func (b *DummyToolCall) NodeName() string           { return b.name }
func (b *DummyToolCall) TextInPort() chan<- string  { return b.in }
func (b *DummyToolCall) TextOutPort() <-chan string { return b.out.Subscribe() }
func (b *DummyToolCall) RequiredInputs() []ports.Kind {
	return []ports.Kind{ports.KindText}
}

func (b *DummyToolCall) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.meter.usage.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.meter.bill(adapters.Usage{Model: "mock-tool", InputTokens: int64(40 + len(v)), OutputTokens: 30})
				b.out.Send(fmt.Sprintf("[%s] %s", b.role, v))
			}
		}
	}()
}

// DummyThreaded stands in for a call on a continuing conversation. It reports
// its turn number so a wiring test can see the thread is actually continuous.
type DummyThreaded struct {
	name, role string

	// Guarded because ExportState is called from the caller's goroutine while
	// the block's own loop is writing — the checkpoint never stops the graph.
	mu   sync.Mutex
	turn int
	// thread stands in for the provider's continuation token — the thing that
	// makes this role threaded, and the thing a resumed session needs back.
	thread string

	in    ports.TextInput
	out   ports.TextBroadcast
	meter meter
}

func (b *DummyThreaded) ExportState() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.thread
}

func (b *DummyThreaded) RestoreState(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.thread = s
	// A restored thread continues its numbering rather than restarting, which
	// is how a test can tell a real resume from a fresh session.
	if n, err := strconv.Atoi(strings.TrimPrefix(s, "resp-")); err == nil {
		b.turn = n
	}
}

func NewDummyThreaded(name, role string) *DummyThreaded {
	return &DummyThreaded{name: name, role: role, in: make(ports.TextInput, 16), meter: meter{name: name}}
}

// UsageOutPort reports what the turns have cost.
func (b *DummyThreaded) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

func (b *DummyThreaded) NodeName() string             { return b.name }
func (b *DummyThreaded) TextInPort() chan<- string    { return b.in }
func (b *DummyThreaded) TextOutPort() <-chan string   { return b.out.Subscribe() }
func (b *DummyThreaded) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

func (b *DummyThreaded) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.meter.usage.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				b.meter.bill(adapters.Usage{Model: "mock-story", InputTokens: int64(400 + len(v)), OutputTokens: 350})
				b.mu.Lock()
				b.turn++
				turn := b.turn
				b.thread = fmt.Sprintf("resp-%d", turn)
				b.mu.Unlock()
				b.out.Send(fmt.Sprintf("[%s turn %d] %s", b.role, turn, v))
			}
		}
	}()
}

// DummyExtraction is the block with two text outputs, which is the case that
// forced SecondaryTextOut: Outline emits a plot line and an image prompt, both
// strings, and they must not be interchangeable at a call site.
type DummyExtraction struct {
	name      string
	in        ports.TextInput
	propsIn   ports.PropsInput
	out       ports.TextBroadcast
	secondary ports.TextBroadcast
	props     ports.PropsBroadcast

	mu      sync.Mutex
	current ports.PropMap
	turn    int
	meter   meter
}

func NewDummyExtraction(name string) *DummyExtraction {
	return &DummyExtraction{
		name:    name,
		in:      make(ports.TextInput, 16),
		propsIn: make(ports.PropsInput, 16),
		current: ports.PropMap{},
		meter:   meter{name: name},
	}
}

// UsageOutPort reports what the outlines have cost.
func (b *DummyExtraction) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

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
		defer func() { b.out.Close(); b.secondary.Close(); b.props.Close(); b.meter.usage.Close() }()

		// A required input is required: process nothing until the first
		// snapshot of the current values has arrived. Without this the block
		// can handle a player action before the seeded values reach it, decide
		// from defaults, and overwrite the real values on the way out.
		select {
		case <-ctx.Done():
			return
		case initial, ok := <-b.propsIn:
			if !ok {
				return
			}
			b.mu.Lock()
			for k, v := range initial {
				b.current[k] = v
			}
			b.mu.Unlock()
		}

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

				b.meter.bill(adapters.Usage{Model: "mock-story", InputTokens: int64(600 + len(v)), OutputTokens: 120})
				b.out.Send(fmt.Sprintf("[outline] world reacts to: %s", v))
				b.secondary.Send("[image-prompt] dim stone bridge, torchlight")
				b.props.Send(ports.PropMap{"Health": health, "Turn": fmt.Sprint(turn)})
			}
		}
	}()
}

type DummyImage struct {
	Switch
	name  string
	in    ports.TextInput
	out   ports.ImageBroadcast
	meter meter
}

func NewDummyImage(name string) *DummyImage {
	return &DummyImage{name: name, in: make(ports.TextInput, 16), meter: meter{name: name}}
}

// UsageOutPort reports what the pictures have cost.
func (b *DummyImage) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

func (b *DummyImage) NodeName() string                     { return b.name }
func (b *DummyImage) TextInPort() chan<- string            { return b.in }
func (b *DummyImage) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }
func (b *DummyImage) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindText} }

func (b *DummyImage) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.meter.usage.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				if !b.Active() {
					continue
				}
				b.meter.bill(adapters.Usage{Model: "mock-image", InputTokens: int64(len(v)), Images: 1})
				b.out.Send(ports.ImageData("<png of " + strings.TrimPrefix(v, "[image-prompt] ") + ">"))
			}
		}
	}()
}

type DummyTTS struct {
	Switch
	name  string
	in    ports.TextInput
	out   ports.AudioBroadcast
	meter meter
}

func NewDummyTTS(name string) *DummyTTS {
	return &DummyTTS{name: name, in: make(ports.TextInput, 16), meter: meter{name: name}}
}

// UsageOutPort reports what the narration has cost.
func (b *DummyTTS) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

func (b *DummyTTS) NodeName() string                      { return b.name }
func (b *DummyTTS) TextInPort() chan<- string             { return b.in }
func (b *DummyTTS) AudioOutPort() <-chan ports.AudioChunk { return b.out.Subscribe() }
func (b *DummyTTS) RequiredInputs() []ports.Kind          { return []ports.Kind{ports.KindText} }

func (b *DummyTTS) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.meter.usage.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				if !b.Active() {
					continue
				}
				// About fifteen characters a second, as narration is read.
				b.meter.bill(adapters.Usage{Model: "mock-speech", AudioSeconds: float64(len(v)) / 15})
				b.out.Send(ports.AudioChunk("<narration of " + v + ">"))
			}
		}
	}()
}
