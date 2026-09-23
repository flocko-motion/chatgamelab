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
	guardrail string
	scenario  string

	maxInARow   int
	consecutive int

	in    ports.TextInput
	out   ports.TextBroadcast
	state ports.StateBroadcast
	usage ports.UsageBroadcast

	spent adapters.Usage

	Flags ports.TextInput
}

func NewObserver(name string, tool adapters.Tool, guardrail, scenario string, maxInARow int) *Observer {
	return &Observer{
		name:      name,
		tool:      tool,
		guardrail: guardrail,
		scenario:  scenario,
		maxInARow: maxInARow,
		in:        make(ports.TextInput, 32),
		Flags:     make(ports.TextInput, 64),
	}
}

func (b *Observer) NodeName() string             { return b.name }
func (b *Observer) TextInPort() chan<- string    { return b.in }
func (b *Observer) TextOutPort() <-chan string   { return b.out.Subscribe() }
func (b *Observer) RequiredInputs() []ports.Kind { return []ports.Kind{ports.KindText} }

// UsageOutPort reports what this block has spent so far.
func (b *Observer) UsageOutPort() <-chan ports.Usage { return b.usage.Subscribe() }

// StateOutPort reports each judgement. An observer is input-driven, so it
// flickers once per line rather than staying busy.
func (b *Observer) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

func (b *Observer) systemPrompt() string {
	return strings.Join([]string{
		"You judge one line spoken by a character in a game.",
		"Answer VIOLATION followed by a short reason, or OK.",
		"Youth-protection constraint (set by the platform, not by the game author): " + b.guardrail,
		"Scenario (set by the game author): " + b.scenario,
		"Judge three things: does the line breach the constraint, is it sycophantic,",
		"and has the character conceded something the scenario says they must not.",
	}, "\n")
}

func (b *Observer) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.state.Close(); b.usage.Close() }()
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		for {
			select {
			case <-ctx.Done():
				return
			case line, open := <-b.in:
				if !open {
					return
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

func (b *Observer) flag(s string) {
	select {
	case b.Flags <- s:
	default:
	}
}

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
