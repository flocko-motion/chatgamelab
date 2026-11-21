package genre

import (
	"testing"

	"engine/internal/ports"
)

// Every genre's drawing is placed by hand, so a block added to the wiring
// must be given a place in the same change, and a removed one taken out.
func TestGenreLayoutsPlaceEveryBlock(t *testing.T) {
	cases := []struct {
		name   string
		wiring *Wiring
		layout ports.Layout
	}{
		{"adventure", adventure(t, AdventureConfig{}), adventureLayout},
		{"npc-live", mustNPCLive(t, mockNPCConfig("Suitable for a 13-year-old.")), npcLiveLayout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nodes := map[string]bool{}
			for _, node := range c.wiring.Graph.Topology().Nodes {
				// Failures are not drawn, since something can always go
				// wrong, and outputs hang off the block that feeds them.
				if node.Role == "failure" || node.Role == "sink" {
					continue
				}
				nodes[node.Name] = true
				if node.Place == nil {
					t.Errorf("%s has no place in the layout", node.Name)
				}
			}
			for name := range c.layout {
				if !nodes[name] {
					t.Errorf("the layout places %s, which the wiring does not have", name)
				}
			}
		})
	}
}
