package blocks

import (
	"context"
	"strings"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// LiveSession is one full-duplex speech-to-speech conversation. Player speech
// arrives on the audio port, the observer's correction on the text port, and
// both audio and its transcript go out.
//
// There is one of these, not a dummy and a real one: what varies is the adapter
// it holds.
type LiveSession struct {
	name string
	live adapters.Live
	cfg  adapters.LiveConfig

	mic   chan ports.AudioChunk
	steer chan string

	audio ports.AudioBroadcast
	text  ports.TextBroadcast
}

func NewLiveSession(name string, live adapters.Live, cfg adapters.LiveConfig) *LiveSession {
	return &LiveSession{
		name:  name,
		live:  live,
		cfg:   cfg,
		mic:   make(chan ports.AudioChunk, 32),
		steer: make(chan string, 16),
	}
}

func (b *LiveSession) NodeName() string                      { return b.name }
func (b *LiveSession) AudioInPort() chan<- ports.AudioChunk  { return b.mic }
func (b *LiveSession) TextInPort() chan<- string             { return b.steer }
func (b *LiveSession) AudioOutPort() <-chan ports.AudioChunk { return b.audio.Subscribe() }
func (b *LiveSession) TextOutPort() <-chan string            { return b.text.Subscribe() }
func (b *LiveSession) RequiredInputs() []ports.Kind          { return []ports.Kind{ports.KindAudio} }

func (b *LiveSession) Start(ctx context.Context) {
	conn, err := b.live.Open(ctx, b.cfg)
	if err != nil {
		b.text.Send("[engine] live session failed to open: " + err.Error())
		b.audio.Close()
		b.text.Close()
		return
	}

	// Inbound: whatever the model produces, split onto the two output ports.
	go func() {
		defer func() { b.audio.Close(); b.text.Close(); conn.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case e, open := <-conn.Events():
				if !open {
					return
				}
				switch e.Kind {
				case adapters.EventAudio:
					b.audio.Send(ports.AudioChunk(e.Audio))
				case adapters.EventTranscript:
					b.text.Send(e.Text)
				case adapters.EventError:
					b.text.Send("[engine] live error: " + e.Err.Error())
				}
			}
		}
	}()

	// Outbound: player speech, and the observer's corrections.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case a, open := <-b.mic:
				if !open {
					return
				}
				_ = conn.Send(a)
			case s, open := <-b.steer:
				if !open {
					return
				}
				_ = conn.Instruct(strings.TrimPrefix(s, steerPrefix))
			}
		}
	}()
}
