package blocks

import (
	"context"

	"engine/internal/ports"
)

// FailureDrop takes a call's failure and does nothing with it: for a call the
// game goes on without, whose failure the player need not hear of. The failed
// call's own state report already says what went wrong, for whoever looks.
type FailureDrop struct {
	name string
	in   ports.FailureInput
}

func NewFailureDrop(name string) *FailureDrop {
	return &FailureDrop{name: name, in: make(ports.FailureInput, 16)}
}

func (b *FailureDrop) NodeName() string                    { return b.name }
func (b *FailureDrop) FailureInPort() chan<- ports.Failure { return b.in }
func (b *FailureDrop) RequiredInputs() []ports.Kind        { return []ports.Kind{ports.KindFailure} }

func (b *FailureDrop) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-b.in:
				if !open {
					return
				}
			}
		}
	}()
}
