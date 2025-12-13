package stream

import (
	"cgl/obj"
	"context"
	"sync"

	"github.com/google/uuid"
)

// Stream represents an active streaming response
type Stream struct {
	MessageID uuid.UUID
	Chunks    chan obj.GameSessionMessageChunk
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
		Chunks:    make(chan obj.GameSessionMessageChunk, 100), // buffered channel
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
func (s *Stream) Send(chunk obj.GameSessionMessageChunk) {
	select {
	case s.Chunks <- chunk:
	default:
		// Buffer full, drop chunk (shouldn't happen with reasonable buffer)
	}
}

// SendText sends a text chunk, with isDone=true for the final chunk
func (s *Stream) SendText(text string, isDone bool) {
	s.Send(obj.GameSessionMessageChunk{Text: text, TextDone: isDone})
}

// SendError signals an error
func (s *Stream) SendError(err string) {
	s.Send(obj.GameSessionMessageChunk{Error: err})
}

// SendImage sends an image chunk, with isDone=true for the final image
func (s *Stream) SendImage(data []byte, isDone bool) {
	s.Send(obj.GameSessionMessageChunk{ImageData: data, ImageDone: isDone})
}
