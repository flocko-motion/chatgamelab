package genre

import (
	"engine/internal/blocks"
	"engine/internal/ports"
)

// Wiring is what every genre constructor returns: the graph, the player-input
// handles that genre accepts, and its output streams keyed by name. The public
// engine package drives a session through this without knowing which genre it
// got.
type Wiring struct {
	Graph *ports.Graph

	// Say and Speak are nil on a genre that doesn't take that input kind, or on
	// a scripted wiring where a dummy source drives the graph instead.
	Say   func(string)
	Speak func([]byte)

	// Sinks are the wired output blocks, which are exactly this genre's event
	// schema.
	Sinks map[string]chan string

	// Observer is set only by genres that run one, so a test can read what it
	// flagged.
	Observer *blocks.Observer
}
