package ports

import (
	"fmt"
	"strings"
	"sync"
)

// The last few values through a node, kept so a reader can ask what a block
// actually did rather than inferring it from the conversation. Small on
// purpose: this is a window, not a log, and the event stream is where a full
// record belongs.
const (
	recentPerPort = 3
	valueLimit    = 400
)

// Sample is one value seen on an edge.
type Sample struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	// Peer is the block at the other end, which is what makes a sample
	// meaningful: "text from rephrase" says more than "text".
	Peer string `json:"peer"`
}

// NodeDetail is everything the graph knows about one node.
type NodeDetail struct {
	Name    string   `json:"name"`
	Role    string   `json:"role"`
	Type    string   `json:"type"`
	Inputs  []Sample `json:"inputs"`
	Outputs []Sample `json:"outputs"`
}

type inspector struct {
	mu      sync.Mutex
	inputs  map[any][]Sample
	outputs map[any][]Sample
}

func newInspector() *inspector {
	return &inspector{inputs: map[any][]Sample{}, outputs: map[any][]Sample{}}
}

func (i *inspector) observe(src, dst any, kind Kind, value any) {
	sample := Sample{Kind: kind.String(), Value: describe(value)}

	i.mu.Lock()
	defer i.mu.Unlock()

	out := sample
	out.Peer = NodeName(dst)
	i.outputs[src] = appendRecent(i.outputs[src], out)

	in := sample
	in.Peer = NodeName(src)
	i.inputs[dst] = appendRecent(i.inputs[dst], in)
}

func appendRecent(samples []Sample, sample Sample) []Sample {
	samples = append(samples, sample)
	if len(samples) > recentPerPort {
		samples = samples[len(samples)-recentPerPort:]
	}
	return samples
}

// describe renders a value for a human. Media is summarised rather than
// carried: a reader wants to know that audio went past and how much of it, and
// a modal is no place for a megabyte of PCM.
func describe(value any) string {
	switch typed := value.(type) {
	case string:
		return truncate(typed)
	case AudioChunk:
		return fmt.Sprintf("audio, %d bytes", len(typed))
	case ImageData:
		return fmt.Sprintf("image, %d bytes", len(typed))
	case PropMap:
		return truncate(fmt.Sprint(map[string]string(typed)))
	case State:
		return string(typed.Phase)
	case Usage:
		return fmt.Sprintf("%s", typed.Model)
	default:
		return truncate(fmt.Sprint(value))
	}
}

func truncate(s string) string {
	if len(s) <= valueLimit {
		return s
	}
	return s[:valueLimit] + "…"
}

// Inspect reports what the graph knows about one node, or false if the wiring
// has no such block.
func (g *Graph) Inspect(name string) (NodeDetail, bool) {
	for _, n := range g.nodes {
		if NodeName(n) != name {
			continue
		}

		g.inspector.mu.Lock()
		// Empty rather than nil: a sink has no outputs and a source has no
		// inputs, and a client that reads a list should be handed a list.
		detail := NodeDetail{
			Name:    name,
			Role:    g.role(n),
			Type:    strings.TrimPrefix(fmt.Sprintf("%T", n), "*blocks."),
			Inputs:  append(make([]Sample, 0, recentPerPort), g.inspector.inputs[n]...),
			Outputs: append(make([]Sample, 0, recentPerPort), g.inspector.outputs[n]...),
		}
		g.inspector.mu.Unlock()
		return detail, true
	}
	return NodeDetail{}, false
}
