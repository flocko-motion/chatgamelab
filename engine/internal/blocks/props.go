package blocks

import (
	"context"
	"encoding/json"
	"maps"
	"sync"

	"engine/internal/ports"
)

// PropsStore holds the session's current property values and re-emits them
// whenever they change. It exists so a block that needs the current values gets
// them on an edge rather than trusting a provider-side thread to remember, and
// so there is one place to seed them from the spec or correct them from outside.
type PropsStore struct {
	name string

	mu      sync.Mutex
	current ports.PropMap

	in  ports.PropsInput
	out ports.PropsBroadcast
}

func NewPropsStore(name string, initial map[string]string) *PropsStore {
	seed := ports.PropMap{}
	maps.Copy(seed, initial)
	return &PropsStore{name: name, current: seed, in: make(ports.PropsInput, 16)}
}

func (b *PropsStore) NodeName() string                   { return b.name }
func (b *PropsStore) PropsInPort() chan<- ports.PropMap  { return b.in }
func (b *PropsStore) PropsOutPort() <-chan ports.PropMap { return b.out.Subscribe() }

// ExportState carries the values themselves, not a handle to them: unlike a
// thread token, nothing on a provider's side remembers a player's health.
func (b *PropsStore) ExportState() string {
	current := b.Current()
	if len(current) == 0 {
		return ""
	}
	blob, err := json.Marshal(current)
	if err != nil {
		return ""
	}
	return string(blob)
}

func (b *PropsStore) RestoreState(s string) {
	var restored ports.PropMap
	if err := json.Unmarshal([]byte(s), &restored); err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = restored
}

// Current is what the store holds right now, for a checkpoint or a test.
func (b *PropsStore) Current() ports.PropMap {
	b.mu.Lock()
	defer b.mu.Unlock()
	return maps.Clone(b.current)
}

func (b *PropsStore) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		// Always publish once at start, even when empty. A block whose props
		// input is required waits for this first snapshot, so withholding it
		// when there is nothing to seed would deadlock that block instead of
		// telling it the values are known and empty.
		b.out.Send(b.Current())
		for {
			select {
			case <-ctx.Done():
				return
			case update, open := <-b.in:
				if !open {
					return
				}
				b.mu.Lock()
				maps.Copy(b.current, update)
				snapshot := maps.Clone(b.current)
				b.mu.Unlock()
				b.out.Send(snapshot)
			}
		}
	}()
}
