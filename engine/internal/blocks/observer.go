package blocks

import (
	"context"
	"strings"

	"engine/internal/adapters"
	"engine/internal/ports"
)

const steerPrefix = "[steer] "

// Observer watches the character's own transcript and steers it back when it
// drifts. It is the cycle in NPC-Live's graph: its output feeds the block whose
// output it reads.
//
// Two prompt slots with different provenance. Guardrail is the platform's
// resolved youth-protection constraint and is not authored by a game designer;
// Scenario is the designer's own text — the setting, the character, and what
// they must not concede.
type Observer struct {
	name      string
	tool      adapters.Tool
	judge     string
	guardrail string
	scenario  string

	maxInARow   int
	consecutive int
	// mute keeps the block on the graph while it does nothing: it reads its
	// input, calls no model and emits nothing. The voice path is worth proving
	// before a second model is added to it, and a node that vanished while
	// muted would make the wiring lie about what the genre contains.
	mute bool

	in    ports.TextInput
	out   ports.TextBroadcast
	flags ports.TextBroadcast
	state ports.StateBroadcast
	usage ports.UsageBroadcast

	spent adapters.Usage
}

// NewObserver takes the judging instructions rather than holding them, because
// what an observer looks for is a property of the genre it serves.
func NewObserver(name string, tool adapters.Tool, judge, guardrail, scenario string, maxInARow int) *Observer {
	return &Observer{
		name:      name,
		tool:      tool,
		judge:     judge,
		guardrail: guardrail,
		scenario:  scenario,
		maxInARow: maxInARow,
		in:        make(ports.TextInput, 32),
	}
}

// Mute stops the block judging anything. It still consumes its input, so
// nothing upstream blocks on a full channel, and it still appears on the graph.
func (b *Observer) Mute() *Observer {
	b.mute = true
	return b
}

func (b *Observer) NodeName() string           { return b.name }
func (b *Observer) TextInPort() chan<- string  { return b.in }
func (b *Observer) TextOutPort() <-chan string { return b.out.Subscribe() }

// SecondaryTextOutPort carries what the observer found, which is a different
// thing from what it does about it: the steering instruction goes to the
// character, and this goes to whoever is watching how the game is played.
//
// It is a port rather than a channel hanging off the struct because an output
// the graph cannot see is an output nothing can validate, nobody can click, and
// no drawing of this genre admits to.
func (b *Observer) SecondaryTextOutPort() <-chan string { return b.flags.Subscribe() }

// RequiredOutputs insists both are wired. A genre taking the steering and
// dropping the findings would be running a guardrail whose results nobody ever
// sees, which is worse than not running one.
func (b *Observer) RequiredOutputs() []ports.Kind { return []ports.Kind{ports.KindText} }
func (b *Observer) RequiredInputs() []ports.Kind  { return []ports.Kind{ports.KindText} }

// UsageOutPort reports what this block has spent so far.
func (b *Observer) UsageOutPort() <-chan ports.Usage { return b.usage.Subscribe() }

// StateOutPort reports each judgement. An observer is input-driven, so it
// flickers once per line rather than staying busy.
func (b *Observer) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// systemPrompt joins what the observer is looking for to what it is looking at.
// The two are kept apart on purpose: the judging instructions are the genre's,
// while the constraint and the scenario belong to the platform and the game
// designer respectively.
func (b *Observer) systemPrompt() string {
	return strings.Join([]string{
		b.judge,
		"Youth-protection constraint (set by the platform, not by the game author): " + b.guardrail,
		"Scenario (set by the game author): " + b.scenario,
	}, "\n")
}

func (b *Observer) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.flags.Close(); b.state.Close(); b.usage.Close() }()
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		for {
			select {
			case <-ctx.Done():
				return
			case line, open := <-b.in:
				if !open {
					return
				}
				if b.mute {
					continue
				}
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
				verdict, used, err := b.tool.Query(ctx, b.systemPrompt(), line)
				b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
				b.report(used)
				if err != nil {
					b.flag("classifier failed: " + err.Error())
					continue
				}
				if !strings.HasPrefix(verdict, "VIOLATION") {
					b.consecutive = 0
					continue
				}
				// Steering produces more output for this same block to judge,
				// so the cycle needs a bound or it can sustain itself.
				if b.consecutive >= b.maxInARow {
					b.flag("giving up after " + itoa(b.consecutive) + " interventions: " + line)
					continue
				}
				b.consecutive++
				b.flag(verdict + " | " + line)
				b.out.Send(steerPrefix + b.guardrail + " " + b.scenario)
			}
		}
	}()
}

// report accumulates and publishes the running total, so a late subscriber sees
// everything spent rather than only what came after it.
func (b *Observer) report(used adapters.Usage) {
	b.spent.Add(used)
	b.usage.Send(ports.Usage{
		Node:              b.name,
		Model:             b.spent.Model,
		InputTokens:       b.spent.InputTokens,
		CachedInputTokens: b.spent.CachedInputTokens,
		OutputTokens:      b.spent.OutputTokens,
		AudioSeconds:      b.spent.AudioSeconds,
	})
}

func (b *Observer) flag(s string) { b.flags.Send(s) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
