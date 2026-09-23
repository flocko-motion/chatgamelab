// Package ports defines the typed edges a genre's wiring is built from.
//
// A port is a Go interface keyed to a data type, never to a role: a tool-call
// block is a TextOut whether it is rephrasing, translating or condensing, which
// is what lets one block type serve many roles.
package ports

import "context"

type (
	AudioChunk []byte
	ImageData  []byte
	PropMap    map[string]string
)

// Source ports hand out a fresh channel per call, so connecting a source twice
// yields two independent streams rather than two consumers racing for one.
type (
	AudioOut         interface{ AudioOutPort() <-chan AudioChunk }
	TextOut          interface{ TextOutPort() <-chan string }
	SecondaryTextOut interface{ SecondaryTextOutPort() <-chan string }
	ImageOut         interface{ ImageOutPort() <-chan ImageData }
	PropsOut         interface{ PropsOutPort() <-chan PropMap }
)

// Phase is what a block is doing. In the init stem a block runs idle → working
// → done once; in the turn loop it oscillates, and a long-lived block like a
// live session stays working for as long as the conversation lasts.
type Phase string

const (
	PhaseIdle    Phase = "idle"
	PhaseWorking Phase = "working"
	PhaseDone    Phase = "done"
)

// State is a block reporting on itself. It names the block because a gate has
// to tell one reporter from another — and because a view drawing the graph
// needs to know which node lit up.
type State struct {
	Node  string `json:"node"`
	Phase Phase  `json:"phase"`
}

type StateOut interface{ StateOutPort() <-chan State }
type StateIn interface{ StateInPort() chan<- State }

type (
	AudioIn interface{ AudioInPort() chan<- AudioChunk }
	TextIn  interface{ TextInPort() chan<- string }
	ImageIn interface{ ImageInPort() chan<- ImageData }
	PropsIn interface{ PropsInPort() chan<- PropMap }
)

type Kind int

const (
	KindAudio Kind = iota
	KindText
	KindImage
	KindProps
	KindState
)

func (k Kind) String() string {
	switch k {
	case KindAudio:
		return "audio"
	case KindText:
		return "text"
	case KindImage:
		return "image"
	case KindProps:
		return "props"
	case KindState:
		return "state"
	}
	return "unknown"
}

// Named lets a block report itself in graph errors and traces. A block that
// doesn't implement it falls back to its Go type name.
type Named interface{ NodeName() string }

// RequiresInputs is what a completeness test checks against: the kinds this
// block cannot run without. Types settle whether an edge is legal; this settles
// whether the graph is finished.
type RequiresInputs interface{ RequiredInputs() []Kind }

// Starter is implemented by blocks that run their own loop.
type Starter interface{ Start(ctx context.Context) }

// Resumable is implemented by blocks holding provider-side state that must
// survive a restart — a threaded call's continuation token, a live session's
// conversation id. The engine never interprets the value; it stores and returns
// it.
//
// State is per block rather than per session: a genre may wire two independent
// threads, and one field could not hold both.
//
// Both methods are called from the caller's goroutine while the block's own
// loop keeps running — a checkpoint never stops the graph — so an implementation
// must guard whatever it reports.
type Resumable interface {
	ExportState() string
	RestoreState(string)
}

// Gatekeeper is a node that waits for every block feeding it to report done
// before the game may begin. It is where the init stem joins the game loop.
//
// The graph tells it how many state edges lead in, because only the graph knows
// the wiring.
type Gatekeeper interface {
	ExpectReporters(n int)
}
