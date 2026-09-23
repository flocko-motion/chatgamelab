package blocks

import (
	"context"
	"sync"

	"engine/internal/ports"
)

// Gate is where the init stem joins the game loop. Every block that must finish
// before play reports done to it; when the last one has, the gate opens and the
// player may act.
//
// What gates is a wiring decision, not a policy: a block whose result the first
// turn needs is wired here, and one that can land late — a portrait, say — is
// not, so a slow picture never holds up a conversation.
type Gate struct {
	name string
	// initPrompt is sent once the gate opens, which is how a genre starts the
	// game rather than waiting to be spoken to. Empty means the gate only
	// releases and says nothing.
	initPrompt string

	in    ports.StateInput
	state ports.StateBroadcast
	text  ports.TextBroadcast

	mu       sync.Mutex
	expected int
	// Tracked by name, not by message: a block in the turn loop works
	// repeatedly, and one busy reporter must not stand in for a silent one.
	worked   map[string]bool
	finished map[string]bool
	open     bool
	ready    chan struct{}
}

// NewGate builds the join between the init stem and the turn loop. The init
// prompt is the first message it sends when it opens — for a live genre, the
// cue that makes the character greet whoever arrived.
func NewGate(name, initPrompt string) *Gate {
	return &Gate{
		name:       name,
		initPrompt: initPrompt,
		in:         make(ports.StateInput, 16),
		worked:     map[string]bool{},
		finished:   map[string]bool{},
		ready:      make(chan struct{}),
	}
}

func (b *Gate) NodeName() string                { return b.name }
func (b *Gate) StateInPort() chan<- ports.State { return b.in }

// TextOutPort carries the init prompt. Wiring it is what turns "the game may
// begin" into "the game begins".
func (b *Gate) TextOutPort() <-chan string { return b.text.Subscribe() }

// StateOutPort is how the gate releases the game. It reports working while it
// waits and ready once it opens, so whatever must not act early listens for
// that rather than being told out of band.
func (b *Gate) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// ExpectReporters is called by the graph, which is the only thing that knows how
// many blocks were wired to report here.
func (b *Gate) ExpectReporters(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.expected = n
	if n == 0 {
		b.openLocked()
	}
}

// Ready closes once every gating block has reported done. A genre with no
// preparation opens immediately, so a caller never has to special-case one.
func (b *Gate) Ready() <-chan struct{} { return b.ready }

func (b *Gate) IsOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.open
}

func (b *Gate) openLocked() {
	if b.open {
		return
	}
	b.open = true
	close(b.ready)
	go func() {
		// Released first, so a player who was already typing is not beaten to
		// the conversation by the character's greeting.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		if b.initPrompt != "" {
			b.text.Send(b.initPrompt)
		}
	}()
}

func (b *Gate) Start(ctx context.Context) {
	// Waiting is work, and showing it as such is what makes a preparation phase
	// visible on a graph rather than a pause nobody can account for.
	b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
	if b.IsOpen() {
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case state, alive := <-b.in:
				if !alive {
					return
				}

				b.mu.Lock()
				switch state.Phase {
				case ports.PhaseWorking:
					b.worked[state.Node] = true
				case ports.PhaseReady:
					// Ready only counts once the block has actually worked;
					// every block reports ready at startup, before it has done
					// anything.
					if b.worked[state.Node] {
						b.finished[state.Node] = true
					}
				}
				if len(b.finished) >= b.expected {
					b.openLocked()
				}
				open := b.open
				b.mu.Unlock()
				if open {
					return
				}
			}
		}
	}()
}
