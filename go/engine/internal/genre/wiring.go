package genre

import (
	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// Wiring is what every genre returns, so the engine drives any genre alike.
type Wiring struct {
	Graph *ports.Graph

	// Prompts are the genre's after spec overrides, so a view can show them.
	Prompts Prompts

	// Say and Speak are nil where a genre lacks that input or a dummy drives the graph.
	Say   func(string)
	Speak func([]byte)

	// Inputs is how the game is played, stated by the genre; clients pick controls from it.
	Inputs []ports.InputMode

	// Imagery is one picture per session or per turn, which decides client placement.
	Imagery ports.Imagery

	// Observer is set only by genres that run one, so tests can read its flags.
	Observer *blocks.Observer

	// Gate opens once every block the first turn depends on reports done.
	Gate *blocks.Gate

	// Live is set by genres whose browser connects directly, for a transport to broker.
	Live *blocks.LiveSession

	// Switches are the blocks each toggle turns off and on during play. A toggle
	// with no entry has no block in this genre.
	Switches map[adapters.Toggle][]ports.Switchable
}

// Sinks is the event schema, read off the graph so outputs follow their edges.
func (w *Wiring) Sinks() map[string]chan string { return w.Graph.Sinks() }
