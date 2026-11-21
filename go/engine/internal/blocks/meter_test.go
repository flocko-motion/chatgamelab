package blocks

import (
	"testing"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// A block whose thread switches model mid-game reports each model's own
// total, so a price applies only to what was spent on its model.
func TestMeterTotalsEachModelApart(t *testing.T) {
	m := meter{name: "expand"}
	reports := m.usage.Subscribe()

	m.bill(adapters.Usage{Model: "mock-story", OutputTokens: 10})
	m.bill(adapters.Usage{Model: "mock-realtime", OutputTokens: 3, AudioOutputTokens: 40})
	m.bill(adapters.Usage{Model: "mock-story", OutputTokens: 5})

	want := []ports.Usage{
		{Node: "expand", Model: "mock-story", OutputTokens: 10},
		{Node: "expand", Model: "mock-realtime", OutputTokens: 3, AudioOutputTokens: 40},
		{Node: "expand", Model: "mock-story", OutputTokens: 15},
	}
	for i, w := range want {
		if got := <-reports; got != w {
			t.Errorf("report %d = %+v, want %+v", i, got, w)
		}
	}
}
