package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Graph is a genre's wiring of inputs, blocks and sinks; cycles are allowed (NPC-Live feeds back on purpose).
type Graph struct {
	Name      string
	inspector *inspector
	nodes     []any
	// shared is state several blocks share, such as a thread.
	shared []Shared
	edges  []Edge
	pumps  []func(context.Context)
	// layout is where each block is drawn, by name.
	layout Layout
	// journal is handed to every node that notes into it, at Start.
	journal *Journal
}

// VoiceMaker is a block that makes a voice of its own for the game, kept on
// the key that pays, rather than speak in a provider's built-in voice.
type VoiceMaker interface {
	MakesVoice()
}

// MakesVoices reports whether a block of the graph makes a voice of its own.
func (g *Graph) MakesVoices() bool {
	for _, n := range g.nodes {
		if _, ok := n.(VoiceMaker); ok {
			return true
		}
	}
	return false
}

// setting is a field of the game as a block reads it, for the drawing: the
// block holds the value from the start, so nothing flows.
type setting struct {
	name, field string
}

func (s *setting) NodeName() string     { return s.name }
func (s *setting) SettingField() string { return s.field }

// Setting draws a field of the game, as its author set it, reaching the block
// that reads it, and keeps its value for the reader to inspect. The block has
// the value already, from its configuration; this only shows where it goes.
func (g *Graph) Setting(dst any, field, value string) {
	src := &setting{name: field + ">" + NodeName(dst), field: field}
	g.connect(src, dst, KindSetting, func(context.Context) {})
	g.inspector.observe(src, dst, KindSetting, value)
}

// UseJournal is where the graph's nodes note what happened; set before Start.
func (g *Graph) UseJournal(j *Journal) { g.journal = j }

type Edge struct {
	From, To any
	Kind     Kind
	// Port names the input it reaches where a block takes two of one kind
	// from the same source, so the drawing keeps them apart; empty: the
	// kind says it.
	Port string
}

func NewGraph(name string) *Graph { return &Graph{Name: name, inspector: newInspector()} }

func NodeName(n any) string {
	if named, ok := n.(Named); ok {
		return named.NodeName()
	}
	return fmt.Sprintf("%T", n)
}

// Track registers a node that has no edges yet, so the graph still starts it.
func (g *Graph) Track(n any) { g.track(n) }

func (g *Graph) track(n any) {
	for _, existing := range g.nodes {
		if existing == n {
			return
		}
	}
	g.nodes = append(g.nodes, n)
}

func (g *Graph) connect(src, dst any, kind Kind, pump func(context.Context)) {
	g.track(src)
	g.track(dst)
	g.edges = append(g.edges, Edge{From: src, To: dst, Kind: kind})
	g.pumps = append(g.pumps, pump)
}

func pumpChan[T any](ctx context.Context, src <-chan T, dst chan<- T, seen func(any)) {
	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-src:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case dst <- v:
			}
			if seen != nil {
				seen(v)
			}
		}
	}
}

func (g *Graph) ConnectAudioOut(src AudioOut, dst AudioIn) {
	stream := src.AudioOutPort()
	g.connect(src, dst, KindAudio, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.AudioInPort(), g.watch(src, dst, KindAudio))
	})
}

func (g *Graph) ConnectTextOut(src TextOut, dst TextIn) {
	stream := src.TextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.TextInPort(), g.watch(src, dst, KindText))
	})
}

func (g *Graph) ConnectSecondaryTextOut(src SecondaryTextOut, dst TextIn) {
	stream := src.SecondaryTextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.TextInPort(), g.watch(src, dst, KindText))
	})
}

// ConnectField wires one field of a structured reply into a block's text
// input: the wire takes the field, so a reply with several feeds several
// blocks, each the part it reads. A field the reply's schema lacks is a
// wiring mistake, caught where the graph is built.
func (g *Graph) ConnectField(src StructuredOut, field string, dst TextIn) {
	g.connectField(src, field, dst, dst.TextInPort())
}

// ConnectFieldToSecondary wires one field of a structured reply into a
// block's second text input.
func (g *Graph) ConnectFieldToSecondary(src StructuredOut, field string, dst SecondaryTextIn) {
	g.connectField(src, field, dst, dst.SecondaryTextInPort())
}

func (g *Graph) connectField(src StructuredOut, field string, dst any, in chan<- string) {
	if !src.HasField(field) {
		panic(fmt.Sprintf("ports: %s has no field %q to wire to %s", NodeName(src), field, NodeName(dst)))
	}
	stream := src.StructuredOutPort()
	fields := make(chan string)
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go func() {
			defer close(fields)
			for reply := range stream {
				if value, ok := fieldOf(reply, field); ok {
					select {
					case fields <- value:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
		go pumpChan(ctx, fields, in, g.watch(src, dst, KindText))
	})
	g.edges[len(g.edges)-1].Port = field
}

// fieldOf is one field of a JSON object as text: a string as it is, any
// other value as JSON; false when the reply has no such field.
func fieldOf(reply, field string) (string, bool) {
	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(reply), &object) != nil {
		return "", false
	}
	raw, ok := object[field]
	if !ok {
		return "", false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, true
	}
	return string(raw), true
}

// ConnectVoiceOut wires a voice, by id, into the conversation that speaks in it.
func (g *Graph) ConnectVoiceOut(src TextOut, dst VoiceIn) {
	stream := src.TextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.VoiceInPort(), g.watch(src, dst, KindText))
	})
	g.edges[len(g.edges)-1].Port = "voice"
}

// ConnectSecondaryToSecondary wires a block's second text output into
// another's second text input.
func (g *Graph) ConnectSecondaryToSecondary(src SecondaryTextOut, dst SecondaryTextIn) {
	stream := src.SecondaryTextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.SecondaryTextInPort(), g.watch(src, dst, KindText))
	})
	g.edges[len(g.edges)-1].Port = "secondary"
}

// ConnectTextOutToSecondary wires text into a block's second text input.
func (g *Graph) ConnectTextOutToSecondary(src TextOut, dst SecondaryTextIn) {
	stream := src.TextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.SecondaryTextInPort(), g.watch(src, dst, KindText))
	})
}

// ConnectOpening wires the cue that opens the game.
func (g *Graph) ConnectOpening(src TextOut, dst OpeningIn) {
	stream := src.TextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.OpeningInPort(), g.watch(src, dst, KindText))
	})
}

// ConnectFailure wires a block's failure output to a block that takes
// failures whole, such as an error sink or the problem sink.
func (g *Graph) ConnectFailure(src FailureOut, dst FailureIn) {
	stream := src.FailureOutPort()
	g.connect(src, dst, KindFailure, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.FailureInPort(), g.watch(src, dst, KindFailure))
	})
}

// ConnectFailureText wires a block's failure output to any block that takes
// text, which receives the input the failed call was given.
func (g *Graph) ConnectFailureText(src FailureOut, dst TextIn) {
	stream := src.FailureOutPort()
	inputs := make(chan string)
	g.connect(src, dst, KindFailure, func(ctx context.Context) {
		go func() {
			defer close(inputs)
			for f := range stream {
				select {
				case inputs <- f.Input:
				case <-ctx.Done():
					return
				}
			}
		}()
		go pumpChan(ctx, inputs, dst.TextInPort(), g.watch(src, dst, KindFailure))
	})
}

// ConnectStreamOut wires streamed text, which only a block taking a stream accepts.
func (g *Graph) ConnectStreamOut(src StreamOut, dst StreamIn) {
	stream := src.StreamOutPort()
	g.connect(src, dst, KindStream, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.StreamInPort(), g.watch(src, dst, KindStream))
	})
}

// Shared is resumable state several blocks share, exported once under its own name.
type Shared interface {
	Named
	Resumable
}

// Share registers shared state, so ExportState and RestoreState reach it.
func (g *Graph) Share(s Shared) { g.shared = append(g.shared, s) }

func (g *Graph) ConnectImageOut(src ImageOut, dst ImageIn) {
	stream := src.ImageOutPort()
	g.connect(src, dst, KindImage, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.ImageInPort(), g.watch(src, dst, KindImage))
	})
}

// ConnectTranscript wires a live conversation's transcript, both speakers.
func (g *Graph) ConnectTranscript(src TranscriptOut, dst TranscriptIn) {
	stream := src.TranscriptOutPort()
	g.connect(src, dst, KindTranscript, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.TranscriptInPort(), g.watch(src, dst, KindTranscript))
	})
}

// ConnectInjectionOut wires text put into a live conversation from the side.
func (g *Graph) ConnectInjectionOut(src InjectionOut, dst InjectionIn) {
	stream := src.InjectionOutPort()
	g.connect(src, dst, KindInjection, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.InjectionInPort(), g.watch(src, dst, KindInjection))
	})
}

// ConnectState wires a block's report on itself, which is how the init stem reaches the gate.
func (g *Graph) ConnectState(src StateOut, dst StateIn) {
	stream := src.StateOutPort()
	g.connect(src, dst, KindState, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.StateInPort(), g.watch(src, dst, KindState))
	})
}

func (g *Graph) ConnectPropsOut(src PropsOut, dst PropsIn) {
	stream := src.PropsOutPort()
	g.connect(src, dst, KindProps, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.PropsInPort(), g.watch(src, dst, KindProps))
	})
}

// Start launches blocks, then pumps; subscribing at Connect time stops an early emitter outrunning consumers.
func (g *Graph) Start(ctx context.Context) {
	// Waiters learn their counts before anything can arrive.
	for _, n := range g.nodes {
		incoming := 0
		for _, e := range g.edges {
			if e.To == n && e.Kind == KindState {
				incoming++
			}
		}
		if gate, ok := n.(Gatekeeper); ok {
			gate.ExpectReporters(incoming)
		}
		if held, ok := n.(Releasable); ok {
			held.ExpectReleases(incoming)
		}
	}

	for _, n := range g.nodes {
		if j, ok := n.(Journaling); ok {
			j.UseJournal(g.journal)
		}
	}
	for _, n := range g.nodes {
		if s, ok := n.(Starter); ok {
			s.Start(ctx)
		}
	}
	for _, p := range g.pumps {
		p(ctx)
	}
}

// Validate checks completeness, which the type system cannot: required edges present, nothing stranded.
func (g *Graph) Validate() error {
	var problems []string

	for _, n := range g.nodes {
		req, ok := n.(RequiresInputs)
		if !ok {
			continue
		}
		for _, kind := range req.RequiredInputs() {
			if !g.hasIncoming(n, kind) {
				problems = append(problems,
					fmt.Sprintf("%s has no incoming %s edge", NodeName(n), kind))
			}
		}
	}

	for _, n := range g.nodes {
		req, ok := n.(RequiresOutputs)
		if !ok {
			continue
		}
		for _, kind := range req.RequiredOutputs() {
			if !g.hasOutgoing(n, kind) {
				problems = append(problems,
					fmt.Sprintf("nothing consumes %s on %s", kind, NodeName(n)))
			}
		}
	}

	for _, n := range g.nodes {
		if g.hasAnyIncoming(n) || g.hasAnyOutgoing(n) {
			continue
		}
		// A lone gate is a genre that needs no preparation.
		if _, isGate := n.(Gatekeeper); isGate {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s is connected to nothing", NodeName(n)))
	}

	for _, n := range g.nodes {
		if _, isGate := n.(Gatekeeper); isGate {
			continue
		}
		if !g.reachable(n) {
			problems = append(problems,
				fmt.Sprintf("%s is unreachable from any source", NodeName(n)))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("graph %q invalid:\n  - %s", g.Name, strings.Join(problems, "\n  - "))
}

// hasIncoming counts a failure edge as text, as one into a text input
// carries the failed call's input.
func (g *Graph) hasIncoming(n any, kind Kind) bool {
	for _, e := range g.edges {
		if e.To == n && (e.Kind == kind || kind == KindText && e.Kind == KindFailure) {
			return true
		}
	}
	return false
}

func (g *Graph) hasOutgoing(n any, kind Kind) bool {
	for _, e := range g.edges {
		if e.From == n && e.Kind == kind {
			return true
		}
	}
	return false
}

func (g *Graph) hasAnyIncoming(n any) bool {
	for _, e := range g.edges {
		if e.To == n {
			return true
		}
	}
	return false
}

func (g *Graph) hasAnyOutgoing(n any) bool {
	for _, e := range g.edges {
		if e.From == n {
			return true
		}
	}
	return false
}

// reachableFromSources walks forward from every node with no inbound edge,
// and from every origin.
func (g *Graph) reachableFromSources() []any {
	seen := map[any]bool{}
	var walk func(n any)
	walk = func(n any) {
		if seen[n] {
			return
		}
		seen[n] = true
		for _, e := range g.edges {
			if e.From == n {
				walk(e.To)
			}
		}
	}
	for _, n := range g.nodes {
		if _, origin := n.(Origin); origin || !g.hasAnyIncoming(n) {
			walk(n)
		}
	}
	out := make([]any, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	return out
}

func (g *Graph) reachable(target any) bool {
	for _, n := range g.reachableFromSources() {
		if n == target {
			return true
		}
	}
	return false
}

// Describe renders the wiring as text.
func (g *Graph) Describe() string {
	var b strings.Builder
	fmt.Fprintf(&b, "graph %q\n", g.Name)
	for _, e := range g.edges {
		fmt.Fprintf(&b, "  %-22s --%s--> %s\n", NodeName(e.From), e.Kind, NodeName(e.To))
	}
	return b.String()
}

// ExportState collects every resumable block's state, keyed by node name.
func (g *Graph) ExportState() map[string]string {
	state := map[string]string{}
	for name, r := range g.resumables() {
		if v := r.ExportState(); v != "" {
			state[name] = v
		}
	}
	return state
}

// resumables are the resumable blocks and the shared state, by name.
func (g *Graph) resumables() map[string]Resumable {
	all := map[string]Resumable{}
	for _, n := range g.nodes {
		if r, ok := n.(Resumable); ok {
			all[NodeName(n)] = r
		}
	}
	for _, s := range g.shared {
		all[s.NodeName()] = s
	}
	return all
}

// RestoreState errors on an unknown name: the wiring changed under a stored session, so it must invalidate.
func (g *Graph) RestoreState(state map[string]string) error {
	known := g.resumables()

	var unknown []string
	for name := range state {
		if _, found := known[name]; !found {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("graph %q has no resumable block named %s; the wiring changed under this session",
			g.Name, strings.Join(unknown, ", "))
	}

	for name, r := range known {
		if v, found := state[name]; found {
			r.RestoreState(v)
		}
	}
	return nil
}

// Sinks is the genre's event schema, read off the wiring so it cannot drift
// from it. Several sinks may share a stream, as error sinks of different
// codes share `error`.
func (g *Graph) Sinks() []Sink {
	var sinks []Sink
	for _, n := range g.nodes {
		if sink, ok := n.(Sink); ok {
			sinks = append(sinks, sink)
		}
	}
	return sinks
}

// Topology is the wiring in drawable form; showing players how a turn is assembled is part of the product.
type Topology struct {
	// Name is the genre; Title is this particular game.
	Name  string         `json:"name"`
	Title string         `json:"title,omitempty"`
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
	// Inputs and Imagery are stated by the genre; a client picks its controls and layout from them.
	Inputs  []string `json:"inputs"`
	Imagery string   `json:"imagery"`
	// Switches are the toggles that turn a block of this wiring off and on
	// during play, such as images where a block makes pictures; a client
	// offers only these.
	Switches []string `json:"switches"`
	// Theme and Language come from the session's spec, for the playback-engine to apply.
	Theme    string `json:"theme,omitempty"`
	Language string `json:"language,omitempty"`
	// Dev lets a developer rearrange the drawing and copy the layout back into code.
	Dev bool `json:"dev,omitempty"`
}

type TopologyNode struct {
	Name string `json:"name"`
	Role string `json:"role"`
	// Field is the game field a setting stands for, such as "voiceStyle".
	Field string `json:"field,omitempty"`
	// Place is where the block is drawn; a block the layout misses has none.
	Place *Place `json:"place,omitempty"`
}

// Place is a block's centre in the drawing, in its own units: x runs left to
// right, y top to bottom.
type Place struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Layout places a genre's blocks by name. A wiring changes only with code, so
// its drawing is placed by hand, in a reading order a person chose.
type Layout map[string]Place

// Arrange gives the graph its drawing.
func (g *Graph) Arrange(layout Layout) { g.layout = layout }

type TopologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
	// Port tells two wires of one kind between the same blocks apart.
	Port string `json:"port,omitempty"`
}

func (g *Graph) Topology() Topology {
	// Inputs is left for the genre to fill.
	t := Topology{Name: g.Name, Inputs: []string{}}
	for _, n := range g.nodes {
		node := TopologyNode{Name: NodeName(n), Role: g.role(n)}
		if s, ok := n.(GameSetting); ok {
			node.Field = s.SettingField()
		}
		if place, ok := g.layout[node.Name]; ok {
			node.Place = &place
		}
		t.Nodes = append(t.Nodes, node)
	}
	for _, e := range g.edges {
		t.Edges = append(t.Edges, TopologyEdge{
			From: NodeName(e.From),
			To:   NodeName(e.To),
			Kind: e.Kind.String(),
			Port: e.Port,
		})
	}
	return t
}

// role ignores control edges: a microphone with a release wired in is still a source.
func (g *Graph) role(n any) string {
	// A gate is drawn as such, whatever its edges: it is where the init stem joins the turn loop.
	if _, isGate := n.(Gatekeeper); isGate {
		return "gate"
	}
	// A field of the game is an input, whatever reads it.
	if _, isSetting := n.(GameSetting); isSetting {
		return "setting"
	}

	in, out, onlyFailures := false, false, true
	for _, e := range g.edges {
		if e.Kind == KindState {
			continue
		}
		if e.To == n {
			in = true
			onlyFailures = onlyFailures && e.Kind == KindFailure
		}
		if e.From == n {
			out = true
		}
	}

	_, player := n.(PlayerInput)
	switch {
	// Only the player's inputs are sources; a block without inputs that
	// makes something itself is drawn as a block.
	case !in && out && player:
		return "source"
	// A sink that only failures reach: where a block's failed call goes.
	case in && !out && onlyFailures:
		return "failure"
	case in && !out:
		return "sink"
	default:
		return "block"
	}
}

// Mermaid renders the wiring as a flowchart for the project's documentation.
func (g *Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("graph TB\n")
	for _, n := range g.nodes {
		name := NodeName(n)
		switch g.role(n) {
		case "source":
			fmt.Fprintf(&b, "  %s[/%s/]\n", id(name), name)
		case "sink", "failure":
			fmt.Fprintf(&b, "  %s[\\%s\\]\n", id(name), name)
		case "gate":
			fmt.Fprintf(&b, "  %s{{%s}}\n", id(name), name)
		default:
			fmt.Fprintf(&b, "  %s([%s])\n", id(name), name)
		}
	}
	for _, e := range g.edges {
		fmt.Fprintf(&b, "  %s -- %s --> %s\n", id(NodeName(e.From)), e.Kind, id(NodeName(e.To)))
	}
	return b.String()
}

func id(name string) string { return strings.ReplaceAll(name, "-", "_") }

// watch records values passing between two blocks, so a block's output can be inspected.
func (g *Graph) watch(src, dst any, kind Kind) func(any) {
	return func(v any) { g.inspector.observe(src, dst, kind, v) }
}

// ObserveUsage merges every block's spending reports into one stream, by introspection like ObserveStates.
func (g *Graph) ObserveUsage(ctx context.Context) <-chan Usage {
	out := make(chan Usage, 128)

	var wg sync.WaitGroup
	for _, n := range g.nodes {
		reporter, ok := n.(UsageOut)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(stream <-chan Usage) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case used, alive := <-stream:
					if !alive {
						return
					}
					select {
					case out <- used:
					case <-ctx.Done():
						return
					}
				}
			}
		}(reporter.UsageOutPort())
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// ObserveStates merges every block's state reports by introspection; the gate takes its selection via edges.
func (g *Graph) ObserveStates(ctx context.Context) <-chan State {
	out := make(chan State, 128)

	var wg sync.WaitGroup
	for _, n := range g.nodes {
		reporter, ok := n.(StateOut)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(stream <-chan State) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case state, alive := <-stream:
					if !alive {
						return
					}
					select {
					case out <- state:
					case <-ctx.Done():
						return
					}
				}
			}
		}(reporter.StateOutPort())
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
