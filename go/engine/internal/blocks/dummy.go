package blocks

import (
	"context"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// The dummies bill invented numbers, priced from the mock's table, so the usage
// view means something without a real model behind it.

type DummyImage struct {
	Switch
	name  string
	in    ports.TextInput
	out   ports.ImageBroadcast
	meter meter
}

func NewDummyImage(name string) *DummyImage {
	return &DummyImage{name: name, in: make(ports.TextInput, 16), meter: meter{name: name}}
}

// UsageOutPort reports what the pictures have cost.
func (b *DummyImage) UsageOutPort() <-chan ports.Usage { return b.meter.usage.Subscribe() }

func (b *DummyImage) NodeName() string                     { return b.name }
func (b *DummyImage) TextInPort() chan<- string            { return b.in }
func (b *DummyImage) ImageOutPort() <-chan ports.ImageData { return b.out.Subscribe() }
func (b *DummyImage) RequiredInputs() []ports.Kind         { return []ports.Kind{ports.KindText} }

func (b *DummyImage) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.meter.usage.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-b.in:
				if !ok {
					return
				}
				if !b.Active() {
					continue
				}
				b.meter.bill(adapters.Usage{Model: "mock-image", InputTokens: int64(len(v)), Images: 1})
				b.out.Send(ports.ImageData("<png of " + v + ">"))
			}
		}
	}()
}
