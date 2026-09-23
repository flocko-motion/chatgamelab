package ports

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Graph is a genre's wiring: nodes of three kinds (inputs, blocks, output
// sinks) joined by typed edges. It is explicitly allowed to contain cycles —
// NPC-Live's observer feeds back into the live session on purpose.
type Graph struct {
	Name  string
	nodes []any
	edges []Edge
	pumps []func(context.Context)
}

type Edge struct {
	From, To any
	Kind     Kind
}

func NewGraph(name string) *Graph { return &Graph{Name: name} }

func NodeName(n any) string {
	if named, ok := n.(Named); ok {
		return named.NodeName()
	}
	return fmt.Sprintf("%T", n)
}

// Track registers a node that has no edges yet, so the graph can still start it
// and tell it what it is waiting for.
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

func pumpChan[T any](ctx context.Context, src <-chan T, dst chan<- T) {
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
		}
	}
}

func (g *Graph) ConnectAudioOut(src AudioOut, dst AudioIn) {
	stream := src.AudioOutPort()
	g.connect(src, dst, KindAudio, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.AudioInPort())
	})
}

func (g *Graph) ConnectTextOut(src TextOut, dst TextIn) {
	stream := src.TextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.TextInPort())
	})
}

func (g *Graph) ConnectSecondaryTextOut(src SecondaryTextOut, dst TextIn) {
	stream := src.SecondaryTextOutPort()
	g.connect(src, dst, KindText, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.TextInPort())
	})
}

func (g *Graph) ConnectImageOut(src ImageOut, dst ImageIn) {
	stream := src.ImageOutPort()
	g.connect(src, dst, KindImage, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.ImageInPort())
	})
}

// ConnectState wires a block's own report on itself. It is what makes the init
// stem visible: everything reporting to the gate runs once before play begins.
func (g *Graph) ConnectState(src StateOut, dst StateIn) {
	stream := src.StateOutPort()
	g.connect(src, dst, KindState, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.StateInPort())
	})
}

func (g *Graph) ConnectPropsOut(src PropsOut, dst PropsIn) {
	stream := src.PropsOutPort()
	g.connect(src, dst, KindProps, func(ctx context.Context) {
		go pumpChan(ctx, stream, dst.PropsInPort())
	})
}

// Start launches every block's own loop, then every edge pump. Subscription
// already happened at Connect time, so a source that emits immediately on Start
// cannot outrun its consumers.
func (g *Graph) Start(ctx context.Context) {
	// A gate has to know what it is waiting for before any of it can arrive.
	for _, n := range g.nodes {
		gate, ok := n.(Gatekeeper)
		if !ok {
			continue
		}
		incoming := 0
		for _, e := range g.edges {
			if e.To == n && e.Kind == KindState {
				incoming++
			}
		}
		gate.ExpectReporters(incoming)
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

// Validate checks what the type system cannot: that the graph is complete.
// Edge legality is settled at compile time; this settles whether every block
// that needs an input has one, and whether anything is stranded.
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
		if g.hasAnyIncoming(n) || g.hasAnyOutgoing(n) {
			continue
		}
		// A gate with nothing to wait for is a genre that needs no preparation,
		// which is legitimate.
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
// Cycles are fine: the seen-set stops the walk, it doesn't reject the graph.
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

// Describe renders the wiring, so a genre's shape can be read without tracing
// the constructor by hand.
func (g *Graph) Describe() string {
	var b strings.Builder
	fmt.Fprintf(&b, "graph %q\n", g.Name)
	for _, e := range g.edges {
		fmt.Fprintf(&b, "  %-22s --%s--> %s\n", NodeName(e.From), e.Kind, NodeName(e.To))
	}
	return b.String()
}

// ExportState collects every resumable block's state, keyed by node name. The
// names come from the wiring, so they are stable across restarts as long as the
// wiring is.
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

// RestoreState hands each block back what it exported. A name with no matching
// block is reported rather than ignored: it means the wiring changed under a
// stored session, which is the case that has to invalidate rather than limp on.
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

// Topology is the wiring in a form something can draw. The engine ships it
// because ChatGameLab is an educational platform: showing how a turn is
// actually assembled is the product, not a debug afterthought.
type Topology struct {
	Name  string         `json:"name"`
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

type TopologyNode struct {
	Name string `json:"name"`
	// Role is what a reader needs to tell a source from a sink at a glance.
	Role string `json:"role"`
}

type TopologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

func (g *Graph) Topology() Topology {
	t := Topology{Name: g.Name}
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

func (g *Graph) role(n any) string {
	in, out := g.hasAnyIncoming(n), g.hasAnyOutgoing(n)
	switch {
	case !in && out:
		return "source"
	case in && !out:
		return "sink"
	default:
		return "block"
	}
}

// Mermaid renders the wiring as a flowchart, which is what the project's own
// documentation already speaks.
func (g *Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("graph LR\n")
	for _, n := range g.nodes {
		name := NodeName(n)
		switch g.role(n) {
		case "source":
			fmt.Fprintf(&b, "  %s[/%s/]\n", id(name), name)
		case "sink":
			fmt.Fprintf(&b, "  %s[\\%s\\]\n", id(name), name)
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

// ObserveUsage merges every block's spending reports into one stream, by the
// same introspection as ObserveStates: what a session costs is a property of
// the whole graph, not of any one edge.
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

// ObserveStates merges every block's state reports into one stream.
//
// This is introspection rather than wiring: the gate consumes selected reports
// through edges because what gates is a deliberate choice, while an observer of
// the whole graph wants all of them and should not need ten edges drawn into a
// collector node to get them.
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
