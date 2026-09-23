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

	in chan ports.Signal

	mu       sync.Mutex
	expected int
	seen     int
	open     bool
	ready    chan struct{}
}

func NewGate(name string) *Gate {
	return &Gate{name: name, in: make(chan ports.Signal, 8), ready: make(chan struct{})}
}

func (b *Gate) NodeName() string                  { return b.name }
func (b *Gate) SignalInPort() chan<- ports.Signal { return b.in }

// ExpectSignals is called by the graph, which is the only thing that knows how
// many blocks were wired to report here.
func (b *Gate) ExpectSignals(n int) {
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
}

func (b *Gate) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, alive := <-b.in:
				if !alive {
					return
				}
				b.mu.Lock()
				b.seen++
				if b.seen >= b.expected {
					b.openLocked()
				}
				done := b.open
				b.mu.Unlock()
				if done {
					return
				}
			}
		}
	}()
}
