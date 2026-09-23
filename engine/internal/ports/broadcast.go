package ports

import "sync"

// Broadcast fans one producer out to independently-buffered consumers. Generics
// are used here and nowhere in the public surface: the plumbing is written once
// per data type, while the ports themselves stay plain interfaces.
type Broadcast[T any] struct {
	mu     sync.Mutex
	subs   []chan T
	closed bool
}

const subBuffer = 64

// Subscribe returns a stream of its own. Two subscribers never steal each
// other's values, which is what keeps a latency-critical edge independent of a
// slow one.
func (b *Broadcast[T]) Subscribe() <-chan T {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan T, subBuffer)
	if b.closed {
		close(ch)
		return ch
	}
	b.subs = append(b.subs, ch)
	return ch
}

func (b *Broadcast[T]) Send(v T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		ch <- v
	}
}

func (b *Broadcast[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
}

type (
	AudioBroadcast  = Broadcast[AudioChunk]
	TextBroadcast   = Broadcast[string]
	ImageBroadcast  = Broadcast[ImageData]
	PropsBroadcast  = Broadcast[PropMap]
	SignalBroadcast = Broadcast[Signal]
)
