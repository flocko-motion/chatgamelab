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
type Resumable interface {
	ExportState() string
	RestoreState(string)
}
