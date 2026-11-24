package blocks

import (
	"context"

	"engine/internal/ports"
)

// OnceText emits one configured line at startup and then closes. It is the init
// stem in its smallest form: whatever hangs off it runs exactly once per
// session, which is how a genre says "make this picture once" with an edge
// instead of with a flag inside the block doing the work.
type OnceText struct {
	name string
	text string

	out   ports.TextBroadcast
	state ports.StateBroadcast

	// once remembers the line went out, across a restart.
	once
	// again sends on a resumed start too (SendAgain).
	again bool
	// field is the game field it holds, when it holds one as it is (AsSetting).
	field string
}

// SendAgain makes it send once on every start, a resumed one too: for a reader
// that keeps what it made of the text (a kept ToolCall), which answers from
// memory and spends nothing. Off, a resumed session stays silent. Set before
// Start.
func (b *OnceText) SendAgain() *OnceText { b.again = true; return b }

// AsSetting names the game field it holds as its author wrote it, such as
// "voiceStyle", so the drawing names it as that setting.
func (b *OnceText) AsSetting(field string) *OnceText { b.field = field; return b }

func NewOnceText(name, text string) *OnceText {
	return &OnceText{name: name, text: text}
}

func (b *OnceText) NodeName() string { return b.name }

// SettingField marks it as drawn like a game setting: a text fixed before
// play, made from what the game's author wrote. Its name says which.
func (b *OnceText) SettingField() string       { return b.field }
func (b *OnceText) TextOutPort() <-chan string { return b.out.Subscribe() }

// StateOutPort reports the one piece of work this block has. A genre that must
// not start play before the stem has run wires this to the gate.
func (b *OnceText) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *OnceText) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close() }()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})

		already := b.done()

		// A resumed session owes the gate its report too, or the game never
		// starts; nothing runs, so it says it was done before. It stays
		// silent, because whatever this line triggered the first time has
		// already been paid for and persisted, unless it sends again for a
		// reader that answers from memory.
		if already {
			if b.again && b.text != "" {
				b.out.Send(b.text)
			}
			b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady, Done: true})
			return
		}
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
		select {
		case <-ctx.Done():
			return
		default:
		}
		// An empty text has nothing to say: whatever reads it is not asked.
		if b.text != "" {
			b.out.Send(b.text)
		}
		b.mark()

		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}()
}
