package stream

import (
	"cgl/obj"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ImageSaver is a function that saves image data to persistent storage
type ImageSaver func(messageID uuid.UUID, imageData []byte) error

// AudioSaver is a function that saves audio data to persistent storage
type AudioSaver func(messageID uuid.UUID, audioData []byte) error

// Stream represents an active streaming response
type Stream struct {
	MessageID  uuid.UUID
	Chunks     chan obj.GameSessionMessageChunk
	ImageSaver ImageSaver
	AudioSaver AudioSaver
	mu         sync.Mutex
	closed     bool
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

const streamTimeout = 5 * time.Minute

// Create creates a new stream for the given message ID
// The stream will automatically be removed after 5 minutes
// ImageSaver is called to persist image data before signaling imageDone
// AudioSaver is called to persist audio data before signaling audioDone
func (r *Registry) Create(ctx context.Context, message *obj.GameSessionMessage, imageSaver ImageSaver, audioSaver AudioSaver) (stream *Stream) {

	r.mu.Lock()
	defer r.mu.Unlock()

	stream = &Stream{
		MessageID:  message.ID,
		Chunks:     make(chan obj.GameSessionMessageChunk, 100), // buffered channel
		ImageSaver: imageSaver,
		AudioSaver: audioSaver,
	}
	r.streams[message.ID] = stream

	// Auto-cleanup after timeout
	go func() {
		time.Sleep(streamTimeout)
		r.Remove(message.ID)
	}()

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
		stream.mu.Lock()
		stream.closed = true
		close(stream.Chunks)
		stream.mu.Unlock()
		delete(r.streams, messageID)
	}
}

// Send sends a chunk to the stream (non-blocking, drops if buffer full or closed)
func (s *Stream) Send(chunk obj.GameSessionMessageChunk) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
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

// SendError signals an error with an optional machine-readable error code
func (s *Stream) SendError(errorCode string, err string) {
	s.Send(obj.GameSessionMessageChunk{Error: err, ErrorCode: errorCode})
}

// SendImage streams partial or final image data to the frontend.
// Partial images (isDone=false) are sent as base64 for immediate preview.
// Final image (isDone=true) is saved to DB and signaled as complete.
func (s *Stream) SendImage(data []byte, isDone bool) {
	if len(data) == 0 {
		return
	}

	if isDone {
		// Save image to DB BEFORE signaling done, so frontend can fetch it
		if s.ImageSaver != nil {
			if err := s.ImageSaver(s.MessageID, data); err != nil {
				// Image save failed, but continue with signaling
			}
		}
		// Signal that image is ready to fetch (frontend uses URL endpoint)
		s.Send(obj.GameSessionMessageChunk{ImageDone: true})
	} else {
		// Send partial image data for WIP preview
		s.Send(obj.GameSessionMessageChunk{ImageData: data})
	}
}

// SendAudio streams partial or final audio data to the frontend.
// Final audio (isDone=true) is saved to DB and signaled as complete.
func (s *Stream) SendAudio(data []byte, isDone bool) {
	if len(data) == 0 && !isDone {
		return
	}

	if isDone {
		// Save audio to DB BEFORE signaling done
		if s.AudioSaver != nil {
			if err := s.AudioSaver(s.MessageID, data); err != nil {
				// Audio save failed, but continue with signaling
			}
		}
		s.Send(obj.GameSessionMessageChunk{AudioDone: true})
	} else {
		// Send partial audio data for streaming playback
		s.Send(obj.GameSessionMessageChunk{AudioData: data})
	}
}
