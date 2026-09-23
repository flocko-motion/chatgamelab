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

// Phase is what a block is doing. Two values are enough: "not working" needs no
// distinction between never-started and finished, and a block in the turn loop
// is never finished anyway.
//
// A block reports its initial phase at startup and then only changes, so a
// reader sees one working and one ready per piece of work.
type Phase string

const (
	PhaseReady   Phase = "ready"
	PhaseWorking Phase = "working"
)

// State is a block reporting on itself. It names the block because a gate has
// to tell one reporter from another — and because a view drawing the graph
// needs to know which node lit up.
type State struct {
	Node  string `json:"node"`
	Phase Phase  `json:"phase"`
}

type StateOut interface{ StateOutPort() <-chan State }

// Usage is a block reporting what it has spent, cumulatively for the session.
// Totals rather than deltas: a dropped event then costs accuracy for an instant
// rather than permanently.
//
// Units differ because providers bill differently, and the engine aggregates by
// model because that is what prices attach to.
type Usage struct {
	Node              string  `json:"node"`
	Model             string  `json:"model"`
	InputTokens       int64   `json:"inputTokens,omitempty"`
	CachedInputTokens int64   `json:"cachedInputTokens,omitempty"`
	OutputTokens      int64   `json:"outputTokens,omitempty"`
	AudioSeconds      float64 `json:"audioSeconds,omitempty"`
	Images            int64   `json:"images,omitempty"`
}

type UsageOut interface{ UsageOutPort() <-chan Usage }
type StateIn interface{ StateInPort() chan<- State }

type (
	AudioIn interface{ AudioInPort() chan<- AudioChunk }
	TextIn  interface{ TextInPort() chan<- string }
	// SecondaryTextIn is the second text input of a block that genuinely has
	// two, since a struct satisfies any given interface only once. A live
	// session is the case: the player's words on one, the observer's correction
	// on the other.
	SecondaryTextIn interface{ SecondaryTextInPort() chan<- string }
	ImageIn         interface{ ImageInPort() chan<- ImageData }
	PropsIn         interface{ PropsInPort() chan<- PropMap }
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

// Gatekeeper is a node that waits for every block feeding it to work and come
// back to ready before the game may begin. It is where the init stem joins the
// game loop.
//
// Waiting for working-then-ready rather than for a "done" message means the
// gate cannot be satisfied by a block that never started. The contract on a
// gating block is therefore to complete that cycle even when it finds it has
// nothing to do — a resumed session skipping its image still reports it.
//
// The graph tells it how many state edges lead in, because only the graph knows
// the wiring.
type Gatekeeper interface {
	ExpectReporters(n int)
}
