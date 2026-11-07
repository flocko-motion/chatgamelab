package ports

import (
	"context"
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
	edges     []Edge
	pumps     []func(context.Context)
}

type Edge struct {
	From, To any
	Kind     Kind
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

func (g *Graph) ConnectImageOut(src ImageOut, dst ImageIn) {
	stream := src.ImageOutPort()
	g.connect(src, dst, KindImage, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.ImageInPort(), g.watch(src, dst, KindImage))
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

func (g *Graph) hasIncoming(n any, kind Kind) bool {
	for _, e := range g.edges {
		if e.To == n && e.Kind == kind {
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

// reachableFromSources walks forward from every node with no inbound edge.
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
		if !g.hasAnyIncoming(n) {
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
	for _, n := range g.nodes {
		if r, ok := n.(Resumable); ok {
			if v := r.ExportState(); v != "" {
				state[NodeName(n)] = v
			}
		}
	}
	return state
}

// RestoreState errors on an unknown name: the wiring changed under a stored session, so it must invalidate.
func (g *Graph) RestoreState(state map[string]string) error {
	known := map[string]bool{}
	for _, n := range g.nodes {
		if _, ok := n.(Resumable); ok {
			known[NodeName(n)] = true
		}
	}

	var unknown []string
	for name := range state {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("graph %q has no resumable block named %s; the wiring changed under this session",
			g.Name, strings.Join(unknown, ", "))
	}

	for _, n := range g.nodes {
		r, ok := n.(Resumable)
		if !ok {
			continue
		}
		if v, found := state[NodeName(n)]; found {
			r.RestoreState(v)
		}
	}
	return nil
}

// Sinks is the genre's event schema, read off the wiring so it cannot drift from it.
func (g *Graph) Sinks() map[string]chan string {
	streams := map[string]chan string{}
	for _, n := range g.nodes {
		if sink, ok := n.(Sink); ok {
			streams[sink.Stream()] = sink.Arrivals()
		}
	}
	return streams
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
}

type TopologyNode struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type TopologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

func (g *Graph) Topology() Topology {
	// Inputs is left for the genre to fill.
	t := Topology{Name: g.Name, Inputs: []string{}}
	for _, n := range g.nodes {
		t.Nodes = append(t.Nodes, TopologyNode{Name: NodeName(n), Role: g.role(n)})
	}
	for _, e := range g.edges {
		t.Edges = append(t.Edges, TopologyEdge{
			From: NodeName(e.From),
			To:   NodeName(e.To),
			Kind: e.Kind.String(),
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

	in, out := false, false
	for _, e := range g.edges {
		if e.Kind == KindState {
			continue
		}
		if e.To == n {
			in = true
		}
		if e.From == n {
			out = true
		}
	}

	switch {
	case !in && out:
		return "source"
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
		case "sink":
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
