package blocks

import (
	"context"
	"encoding/json"

	"engine/internal/ports"
)

// FailureReply takes a call's failure and answers in its place: a fixed
// structured reply, whose fields wires take as they would the call's. For a
// call whose reply others wait for, so they go on at once with what stands in
// for it rather than wait for a reply that will not come.
type FailureReply struct {
	name   string
	fields map[string]string
	in     ports.FailureInput
	out    ports.TextBroadcast
}

// NewFailureReply answers each failure with the fields given.
func NewFailureReply(name string, fields map[string]string) *FailureReply {
	return &FailureReply{name: name, fields: fields, in: make(ports.FailureInput, 16)}
}

func (b *FailureReply) NodeName() string                    { return b.name }
func (b *FailureReply) FailureInPort() chan<- ports.Failure { return b.in }
func (b *FailureReply) RequiredInputs() []ports.Kind        { return []ports.Kind{ports.KindFailure} }

// StructuredOutPort carries the reply, as JSON, once per failure.
func (b *FailureReply) StructuredOutPort() <-chan string { return b.out.Subscribe() }

// HasField says whether the reply has a field, for a wire to take it.
func (b *FailureReply) HasField(name string) bool {
	_, ok := b.fields[name]
	return ok
}

func (b *FailureReply) Start(ctx context.Context) {
	reply, _ := json.Marshal(b.fields)
	go func() {
		defer b.out.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-b.in:
				if !open {
					return
				}
				b.out.Send(string(reply))
			}
		}
	}()
}
