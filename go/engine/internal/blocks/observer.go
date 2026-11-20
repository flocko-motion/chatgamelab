package blocks

import (
	"context"
	"strings"

	"engine/internal/adapters"
	"engine/internal/ports"
)

const steerPrefix = "[steer] "

// Observer steers the character back when its transcript drifts: the cycle in
// NPC-Live's graph. Guardrail is the platform's; Scenario is the designer's.
type Observer struct {
	name      string
	tool      adapters.Tool
	judge     string
	guardrail string
	scenario  string

	maxInARow   int
	consecutive int
	// mute keeps the node on the graph, judging nothing, so the wiring stays truthful.
	mute bool

	in    ports.TextInput
	out   ports.TextBroadcast
	flags ports.TextBroadcast
	state ports.StateBroadcast
	usage ports.UsageBroadcast

	spent adapters.Usage
}

// NewObserver takes the judging instructions because they belong to the genre.
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

// Mute stops judging but still drains input, so nothing upstream blocks.
func (b *Observer) Mute() *Observer {
	b.mute = true
	return b
}

func (b *Observer) NodeName() string           { return b.name }
func (b *Observer) TextInPort() chan<- string  { return b.in }
func (b *Observer) TextOutPort() <-chan string { return b.out.Subscribe() }

// SecondaryTextOutPort carries the findings, for whoever watches play.
func (b *Observer) SecondaryTextOutPort() <-chan string { return b.flags.Subscribe() }

// RequiredOutputs stops a genre running an observer whose output reaches nothing.
func (b *Observer) RequiredOutputs() []ports.Kind { return []ports.Kind{ports.KindText} }
func (b *Observer) RequiredInputs() []ports.Kind  { return []ports.Kind{ports.KindText} }

// UsageOutPort reports what this block has spent so far.
func (b *Observer) UsageOutPort() <-chan ports.Usage { return b.usage.Subscribe() }

// StateOutPort flickers once per judgement.
func (b *Observer) StateOutPort() <-chan ports.State { return b.state.Subscribe() }

// systemPrompt labels each part by its author: genre, platform, game designer.
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
				// Steering yields more lines to judge, so consecutive interventions are bounded.
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

// report publishes the running total, so a late subscriber sees all spending.
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
