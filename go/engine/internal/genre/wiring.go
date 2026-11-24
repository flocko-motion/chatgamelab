package genre

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// Wiring is what every genre returns, so the engine drives any genre alike.
type Wiring struct {
	Graph *ports.Graph

	// Prompts are the genre's after spec overrides and edits during play, as
	// the blocks run on them, so a view can show them. During play they are
	// read through CurrentPrompts and changed through SetPrompt.
	Prompts Prompts
	// Defaults are the genre's own prompts, as they would run in this
	// session, so a view can tell an edited prompt from the genre's.
	Defaults Prompts
	// PromptUses say, by prompt name, which block runs on each prompt and
	// how a changed text reaches it.
	PromptUses map[string]PromptUse
	// Guardrail is the session's youth-protection rule as the blocks keep to
	// it now: one value, which each block that keeps to it words its own way.
	// During play it is read through CurrentGuardrail and changed through
	// SetGuardrail.
	Guardrail string
	// GuardrailUse says which blocks keep to the guardrail and how a changed
	// one reaches them.
	GuardrailUse PromptUse
	// Render fills a prompt's template as the genre does at launch; nil
	// takes the text as it is.
	Render    func(name, text string) (string, error)
	promptsMu sync.Mutex

	// Say and Speak are nil where a genre lacks that input or a dummy drives the graph.
	// Speak refuses what the genre cannot take now, such as speech while
	// voice input is off.
	Say   func(string)
	Speak func([]byte) error

	// Inputs is how the game is played, stated by the genre; clients pick controls from it.
	Inputs []ports.InputMode

	// Imagery is one picture per session or per turn, which decides client placement.
	Imagery ports.Imagery

	// Injectable are the blocks that take an injection during play, by name.
	Injectable map[string]blocks.Injectable

	// Observers are the blocks that follow a conversation from the side, by
	// name, so a reader can see where each stands and a developer can tune it.
	Observers map[string]*blocks.Observer

	// Gate opens once every block the first turn depends on reports done.
	Gate *blocks.Gate

	// Live is set by genres whose browser connects directly, for a transport to broker.
	Live *blocks.LiveSession

	// Partner is a second live conversation beside Live, for a genre of two
	// characters: the browser connects to each, naming its seat (SeatPartner),
	// and the two pause and resume together.
	Partner *blocks.LiveSession

	// Constrained are the blocks whose calls carry the constraints, by name,
	// so their frequency can change while the game runs.
	Constrained map[string]blocks.Constrained

	// OnThread are the blocks that take part in a thread, by name, so a
	// session can read how full their thread is and compact it on demand.
	OnThread map[string]blocks.OnThread

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

// Current is what the blocks run on now: the prompts and the guardrail.
type Current struct {
	Prompts   Prompts
	Guardrail string
}

// PromptUse is where a genre prompt, or the guardrail, goes: the blocks that
// run on it, and how what they run on, with one part changed, reaches them
// again. A thread's System Message is run on by every block on the thread.
type PromptUse struct {
	Blocks []string
	Apply  func(Current)
}

// CurrentGuardrail is the guardrail the blocks keep to now.
func (w *Wiring) CurrentGuardrail() string {
	w.promptsMu.Lock()
	defer w.promptsMu.Unlock()
	return w.Guardrail
}

// SetGuardrail changes the guardrail during play: every block that keeps to
// it takes the new one from its next call; what was done stays.
func (w *Wiring) SetGuardrail(text string) {
	w.promptsMu.Lock()
	defer w.promptsMu.Unlock()
	w.Guardrail = text
	if w.GuardrailUse.Apply != nil {
		w.GuardrailUse.Apply(Current{Prompts: maps.Clone(w.Prompts), Guardrail: text})
	}
}

// CurrentPrompts are the prompts the blocks run on now.
func (w *Wiring) CurrentPrompts() Prompts {
	w.promptsMu.Lock()
	defer w.promptsMu.Unlock()
	return maps.Clone(w.Prompts)
}

// PromptsOf are the names of the prompts a block runs on, sorted.
func (w *Wiring) PromptsOf(block string) []string {
	var names []string
	for name, use := range w.PromptUses {
		if slices.Contains(use.Blocks, block) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// SetPrompt changes one prompt during play: its block runs on the new text
// from its next call. A prompt the genre lacks, or a template that does not
// fill, is refused and changes nothing.
func (w *Wiring) SetPrompt(name, text string) error {
	use, ok := w.PromptUses[name]
	if !ok {
		return fmt.Errorf("no such prompt: %s (this genre has %s)", name, strings.Join(slices.Sorted(maps.Keys(w.PromptUses)), ", "))
	}
	if w.Render != nil {
		rendered, err := w.Render(name, text)
		if err != nil {
			return err
		}
		text = rendered
	}
	w.promptsMu.Lock()
	defer w.promptsMu.Unlock()
	prompts := maps.Clone(w.Prompts)
	prompts[name] = text
	w.Prompts = prompts
	use.Apply(Current{Prompts: maps.Clone(prompts), Guardrail: w.Guardrail})
	return nil
}

// Compact asks a block's thread to compact now.
func (w *Wiring) Compact(block string) error {
	b, ok := w.OnThread[block]
	if !ok {
		return fmt.Errorf("no block %q on a thread (this genre has %s)", block, strings.Join(slices.Sorted(maps.Keys(w.OnThread)), ", "))
	}
	b.Compact()
	return nil
}

// Inject passes an injection to a block that takes one.
func (w *Wiring) Inject(block string, kind ports.InjectionKind, text string) error {
	b, ok := w.Injectable[block]
	if !ok {
		return fmt.Errorf("no block %q takes injections (this genre has %s)", block, strings.Join(slices.Sorted(maps.Keys(w.Injectable)), ", "))
	}
	return b.Inject(kind, text)
}

// Sinks is the event schema, read off the graph so outputs follow their edges.
func (w *Wiring) Sinks() []ports.Sink { return w.Graph.Sinks() }
