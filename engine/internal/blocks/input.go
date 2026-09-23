package blocks

import (
	"context"
	"time"

	"engine/internal/ports"
)

// PlayerInputText is a source node: what the player types.
//
// It is held until the gate releases it, so nothing the player does reaches the
// game before preparation has finished. Words typed during the wait are kept
// and sent on release rather than discarded, because someone who typed while a
// portrait rendered meant to say it.
type PlayerInputText struct {
	name string
	out  ports.TextBroadcast

	release ports.StateInput
	held    holdUntilReleased[string]
}

func NewPlayerInputText(name string) *PlayerInputText {
	return &PlayerInputText{name: name, release: make(ports.StateInput, 4)}
}

func (b *PlayerInputText) NodeName() string                { return b.name }
func (b *PlayerInputText) TextOutPort() <-chan string      { return b.out.Subscribe() }
func (b *PlayerInputText) StateInPort() chan<- ports.State { return b.release }
func (b *PlayerInputText) Say(s string)                    { b.held.submit(s, b.out.Send) }
func (b *PlayerInputText) Close()                          { b.out.Close() }

func (b *PlayerInputText) Start(ctx context.Context) {
	go b.held.wait(ctx, b.release, b.out.Send)
}

// PlayerInputAudio is a source node: what the player speaks.
//
// Held until the gate releases it, like typed input — though speech that
// arrived during the wait is dropped rather than kept, since replaying stale
// audio into a live conversation would be worse than losing it.
type PlayerInputAudio struct {
	name string
	out  ports.AudioBroadcast

	release ports.StateInput
	held    holdUntilReleased[ports.AudioChunk]
}

func NewPlayerInputAudio(name string) *PlayerInputAudio {
	return &PlayerInputAudio{name: name, release: make(ports.StateInput, 4), held: holdUntilReleased[ports.AudioChunk]{discard: true}}
}

func (b *PlayerInputAudio) NodeName() string                      { return b.name }
func (b *PlayerInputAudio) AudioOutPort() <-chan ports.AudioChunk { return b.out.Subscribe() }
func (b *PlayerInputAudio) StateInPort() chan<- ports.State       { return b.release }
func (b *PlayerInputAudio) Speak(a ports.AudioChunk)              { b.held.submit(a, b.out.Send) }
func (b *PlayerInputAudio) Close()                                { b.out.Close() }

func (b *PlayerInputAudio) Start(ctx context.Context) {
	go b.held.wait(ctx, b.release, b.out.Send)
}

// InputScript drives a dummy source. With Interval zero the whole script is
// emitted once at Start; otherwise one entry per tick, optionally looping.
type InputScript struct {
	Lines    []string
	Interval time.Duration
	Repeat   bool
}

// DummyInputText replays a script in place of a player, so a wiring can be
// exercised without anyone typing.
type DummyInputText struct {
	name   string
	script InputScript
	out    ports.TextBroadcast
}

func NewDummyInputText(name string, script InputScript) *DummyInputText {
	return &DummyInputText{name: name, script: script}
}

func (b *DummyInputText) NodeName() string           { return b.name }
func (b *DummyInputText) TextOutPort() <-chan string { return b.out.Subscribe() }

func (b *DummyInputText) Start(ctx context.Context) {
	go replay(ctx, b.script, func(line string) { b.out.Send(line) }, b.out.Close)
}

// DummyInputAudio is the spoken counterpart. Each script line stands in for one
// utterance.
type DummyInputAudio struct {
	name   string
	script InputScript
	out    ports.AudioBroadcast
}

func NewDummyInputAudio(name string, script InputScript) *DummyInputAudio {
	return &DummyInputAudio{name: name, script: script}
}

func (b *DummyInputAudio) NodeName() string                      { return b.name }
func (b *DummyInputAudio) AudioOutPort() <-chan ports.AudioChunk { return b.out.Subscribe() }

func (b *DummyInputAudio) Start(ctx context.Context) {
	go replay(ctx, b.script, func(line string) { b.out.Send(ports.AudioChunk(line)) }, b.out.Close)
}

func replay(ctx context.Context, s InputScript, emit func(string), done func()) {
	defer done()
	if len(s.Lines) == 0 {
		return
	}
	if s.Interval <= 0 {
		for _, line := range s.Lines {
			select {
			case <-ctx.Done():
				return
			default:
			}
			emit(line)
		}
		return
	}
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for i := 0; ; i++ {
		if i >= len(s.Lines) {
			if !s.Repeat {
				return
			}
			i = 0
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			emit(s.Lines[i])
		}
	}
}
