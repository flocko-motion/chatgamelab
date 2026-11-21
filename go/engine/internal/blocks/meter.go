package blocks

import (
	"sync"
	"sync/atomic"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// meter keeps a block's running spend per model and publishes the model's
// total, so a late subscriber sees everything spent rather than only what came
// after it, and a block whose model changes mid-game is priced per model.
type meter struct {
	name  string
	usage ports.UsageBroadcast

	mu    sync.Mutex
	spent map[string]adapters.Usage
}

func (m *meter) bill(used adapters.Usage) {
	m.mu.Lock()
	if m.spent == nil {
		m.spent = map[string]adapters.Usage{}
	}
	total := m.spent[used.Model]
	total.Add(used)
	m.spent[used.Model] = total
	m.mu.Unlock()

	m.usage.Send(ports.Usage{
		Node:                   m.name,
		Model:                  total.Model,
		InputTokens:            total.InputTokens,
		CachedInputTokens:      total.CachedInputTokens,
		OutputTokens:           total.OutputTokens,
		AudioInputTokens:       total.AudioInputTokens,
		CachedAudioInputTokens: total.CachedAudioInputTokens,
		AudioOutputTokens:      total.AudioOutputTokens,
		AudioSeconds:           total.AudioSeconds,
		Images:                 total.Images,
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
