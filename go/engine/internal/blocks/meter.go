package blocks

import (
	"sync"
	"sync/atomic"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// meter keeps a block's running spend and publishes the total, so a late
// subscriber sees everything spent rather than only what came after it.
type meter struct {
	name  string
	usage ports.UsageBroadcast

	mu    sync.Mutex
	spent adapters.Usage
}

func (m *meter) bill(used adapters.Usage) {
	m.mu.Lock()
	m.spent.Add(used)
	total := m.spent
	m.mu.Unlock()

	m.usage.Send(ports.Usage{
		Node:         m.name,
		Model:        total.Model,
		InputTokens:  total.InputTokens,
		OutputTokens: total.OutputTokens,
		AudioSeconds: total.AudioSeconds,
		Images:       total.Images,
	})
}

// Switch turns a block's work off and on during play. Off, the block drops
// what reaches it: it makes nothing new and bills nothing, and what it made
// before stays.
type Switch struct{ off atomic.Bool }

// SetActive turns the block's work on or off.
func (s *Switch) SetActive(on bool) { s.off.Store(!on) }

// Active reports whether the block works on what reaches it.
func (s *Switch) Active() bool { return !s.off.Load() }
