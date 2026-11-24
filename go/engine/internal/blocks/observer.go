package blocks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"engine/internal/ports"
)

// Injectable is a block that takes an injection during play and puts it into
// the conversation it watches.
type Injectable interface {
	Inject(kind ports.InjectionKind, text string) error
}

// Observer watches a live conversation from the side and steers it: its input
// is the conversation's transcript, its output what it puts into the
// conversation. It fits any live pipeline.
//
// It reads its input and lets it go; what it puts in is what a player gives
// it during play.
type Observer struct {
	name string

	in     ports.TextInput
	queued chan ports.Injection
	out    ports.InjectionBroadcast
	state  ports.StateBroadcast
}

// maxInjection keeps an injection inside what a live provider takes in one
// append (GPT-Live: 500 tokens), at a safe four characters a token.
const maxInjection = 2000

var (
	errBlankInjection = errors.New("an injection needs text")
	errLongInjection  = fmt.Errorf("an injection takes at most %d characters", maxInjection)
	errQueueFull      = errors.New("too many injections are still waiting to be passed on")
)

func NewObserver(name string) *Observer {
	return &Observer{
		name:   name,
		in:     make(ports.TextInput, 32),
		queued: make(chan ports.Injection, 8),
	}
}

func (b *Observer) NodeName() string          { return b.name }
func (b *Observer) TextInPort() chan<- string { return b.in }

func (b *Observer) InjectionOutPort() <-chan ports.Injection { return b.out.Subscribe() }

// RequiredInputs and RequiredOutputs hold the observer to its place: it reads
// a conversation and steers one.
func (b *Observer) RequiredInputs() []ports.Kind  { return []ports.Kind{ports.KindText} }
func (b *Observer) RequiredOutputs() []ports.Kind { return []ports.Kind{ports.KindInjection} }

func (b *Observer) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// Inject puts text into the conversation, of a kind that says how the model
// takes it. An unknown kind, blank or overlong text is refused, and so is an
// injection while too many are still waiting to be passed on.
func (b *Observer) Inject(kind ports.InjectionKind, text string) error {
	if !slices.Contains(ports.InjectionKinds, kind) {
		return fmt.Errorf("no injection kind %q (want one of %v)", kind, ports.InjectionKinds)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errBlankInjection
	}
	if utf8.RuneCountInString(text) > maxInjection {
		return errLongInjection
	}
	select {
	case b.queued <- ports.Injection{Kind: kind, Text: text}:
		return nil
	default:
		return errQueueFull
	}
}

func (b *Observer) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close() }()
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-b.in:
				// Read and let go, so the conversation feeding it never waits.
				if !open {
					return
				}
			case in := <-b.queued:
				// A flicker, so the graph shows the injection leaving.
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				b.out.Send(in)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
			}
		}
	}()
}
