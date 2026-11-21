package blocks

import (
	"context"
	"maps"

	"engine/internal/ports"
)

// Parts is what Compose builds a text from: a text from each of its text
// inputs, and the props known so far.
type Parts struct {
	Text, Secondary string
	Props           ports.PropMap
}

// Compose builds a text from what reaches it, with no model call: each text
// on its first input is paired with the next on its second, in the order they
// came, with the latest props, and its build function makes the text from the
// three. Adventure's picture prompt is one: the image prompt and the outline
// of one reply, with what preparation made.
type Compose struct {
	name  string
	build func(Parts) string

	in     ports.TextInput
	second ports.TextInput
	props  ports.PropsInput
	out    ports.TextBroadcast
}

func NewCompose(name string, build func(Parts) string) *Compose {
	return &Compose{
		name:   name,
		build:  build,
		in:     make(ports.TextInput, 16),
		second: make(ports.TextInput, 16),
		props:  make(ports.PropsInput, 16),
	}
}

func (b *Compose) NodeName() string                   { return b.name }
func (b *Compose) TextInPort() chan<- string          { return b.in }
func (b *Compose) SecondaryTextInPort() chan<- string { return b.second }
func (b *Compose) PropsInPort() chan<- ports.PropMap  { return b.props }
func (b *Compose) TextOutPort() <-chan string         { return b.out.Subscribe() }

func (b *Compose) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

func (b *Compose) Start(ctx context.Context) {
	go func() {
		defer b.out.Close()
		var firsts, seconds []string
		props := ports.PropMap{}
		for {
			select {
			case <-ctx.Done():
				return
			case update, open := <-b.props:
				if !open {
					return
				}
				maps.Copy(props, update)
			case text, open := <-b.in:
				if !open {
					return
				}
				firsts = append(firsts, text)
			case text, open := <-b.second:
				if !open {
					return
				}
				seconds = append(seconds, text)
			}
			// Props already waiting are the latest known, whatever order the
			// select took the inputs in.
			drained := false
			for !drained {
				select {
				case update, open := <-b.props:
					if !open {
						return
					}
					maps.Copy(props, update)
				default:
					drained = true
				}
			}
			for len(firsts) > 0 && len(seconds) > 0 {
				b.out.Send(b.build(Parts{Text: firsts[0], Secondary: seconds[0], Props: maps.Clone(props)}))
				firsts, seconds = firsts[1:], seconds[1:]
			}
		}
	}()
}
