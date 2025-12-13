package stream

import (
	"cgl/obj"
	"context"
	"sync"

	"github.com/google/uuid"
)

// Chunk represents a piece of streamed content
type Chunk struct {
	Text  string `json:"text,omitempty"`  // Partial text content
	Done  bool   `json:"done,omitempty"`  // True when stream is complete
	Error string `json:"error,omitempty"` // Error message if failed
}

// Stream represents an active streaming response
type Stream struct {
	MessageID uuid.UUID
	Chunks    chan Chunk
}

// Registry manages active streams
type Registry struct {
	mu      sync.RWMutex
	streams map[uuid.UUID]*Stream
}

var defaultRegistry = &Registry{
	streams: make(map[uuid.UUID]*Stream),
}

// Get returns the default registry
func Get() *Registry {
	return defaultRegistry
}

// Create creates a new stream for the given message ID
func (r *Registry) Create(ctx context.Context, message *obj.GameSessionMessage) (stream *Stream) {

	r.mu.Lock()
	defer r.mu.Unlock()

	stream = &Stream{
		MessageID: message.ID,
		Chunks:    make(chan Chunk, 100), // buffered channel
	}
	r.streams[message.ID] = stream
	return stream
}

// Lookup returns the stream for the given message ID, or nil if not found
func (r *Registry) Lookup(messageID uuid.UUID) *Stream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streams[messageID]
}

// Remove removes the stream for the given message ID
func (r *Registry) Remove(messageID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stream, ok := r.streams[messageID]; ok {
		close(stream.Chunks)
		delete(r.streams, messageID)
	}
}

// Send sends a chunk to the stream (non-blocking, drops if buffer full)
func (s *Stream) Send(chunk Chunk) {
	select {
	case s.Chunks <- chunk:
	default:
		// Buffer full, drop chunk (shouldn't happen with reasonable buffer)
	}
}

// SendText sends a text chunk
func (s *Stream) SendText(text string) {
	s.Send(Chunk{Text: text})
}

// SendDone signals stream completion
func (s *Stream) SendDone() {
	s.Send(Chunk{Done: true})
}

// SendError signals an error
func (s *Stream) SendError(err string) {
	s.Send(Chunk{Error: err, Done: true})
}
