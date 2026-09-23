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

	// Prompts are the genre's own, after any the spec replaced. Carried out of
	// the wiring so a view can show what the session is actually running on.
	Prompts Prompts

	// Say and Speak are nil on a genre that doesn't take that input kind, or on
	// a scripted wiring where a dummy source drives the graph instead.
	Say   func(string)
	Speak func([]byte)

	// Inputs is what a player may supply, which a genre states outright. It is
	// a decision — this is how the game is played — rather than something to be
	// read back off the wiring, and a client picks its controls from it.
	Inputs []ports.InputMode

	// Imagery is how many pictures this genre makes: one for the session, or
	// one per turn. It decides where a client puts them, which is a different
	// question from how many arrive.
	Imagery ports.Imagery

	// Observer is set only by genres that run one, so a test can read what it
	// flagged.
	Observer *blocks.Observer

	// Gate opens when every block the first turn depends on has reported done.
	// A genre with nothing to prepare opens immediately.
	Gate *blocks.Gate

	// Live is set only by genres holding a conversation the player's browser
	// connects to directly, so a transport can broker that connection.
	Live *blocks.LiveSession
}

// Sinks is this genre's event schema: the output blocks it wired, keyed by the
// stream each carries.
//
// A method reading the graph rather than a field somebody fills in, because a
// field can be given a channel that belongs to no node — which is how a block
// once had an output that reached the player while the drawing showed it going
// somewhere else. An output with no edge now goes nowhere, and there is no
// second place for it to go.
func (w *Wiring) Sinks() map[string]chan string { return w.Graph.Sinks() }
