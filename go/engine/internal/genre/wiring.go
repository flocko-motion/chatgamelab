package genre

import (
	"fmt"
	"maps"
	"slices"
	"strings"

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
	// Speak refuses what the genre cannot take now, such as speech while
	// voice input is off.
	Say   func(string)
	Speak func([]byte) error

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

	// Constrained are the blocks whose calls carry the constraints, by name,
	// so their frequency can change while the game runs.
	Constrained map[string]blocks.Constrained

	// Switches are the blocks each toggle turns off and on during play. A toggle
	// with no entry has no block in this genre.
	Switches map[adapters.Toggle][]ports.Switchable
}

// The range of a constraint frequency: 1 carries the constraints with every
// call, 10 with every tenth.
const (
	MinConstraintFrequency = 1
	MaxConstraintFrequency = 10
)

// CheckConstraintFrequency refuses a block whose calls carry no constraints,
// of those constrained, or a frequency out of range: the one check for a
// spec, a wiring and a change during play.
func CheckConstraintFrequency(constrained []string, block string, n int) error {
	if !slices.Contains(constrained, block) {
		return fmt.Errorf("no such constrained block %q (this genre has %s)", block, strings.Join(constrained, ", "))
	}
	if n < MinConstraintFrequency || n > MaxConstraintFrequency {
		return fmt.Errorf("constraint frequency %d: want %d to %d", n, MinConstraintFrequency, MaxConstraintFrequency)
	}
	return nil
}

// SetConstraintFrequency sets how often a block's calls carry the
// constraints, once checked.
func (w *Wiring) SetConstraintFrequency(block string, n int) error {
	if err := CheckConstraintFrequency(slices.Sorted(maps.Keys(w.Constrained)), block, n); err != nil {
		return err
	}
	w.Constrained[block].SetConstraintFrequency(n)
	return nil
}

// Sinks is the event schema, read off the graph so outputs follow their edges.
func (w *Wiring) Sinks() []ports.Sink { return w.Graph.Sinks() }
