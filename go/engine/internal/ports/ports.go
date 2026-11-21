// Package ports defines the typed edges a genre's wiring is built from.
// Ports are keyed to a data type, so one block type can serve many roles.
package ports

import "context"

type (
	AudioChunk []byte
	ImageData  []byte
	PropMap    map[string]string
	// Piece is streamed text: a reply arriving piece by piece. A type of its
	// own, so a block taking whole text is never wired to a stream.
	Piece string
)

// The stream's two markers. A reply ends with PieceClose; PieceDiscard drops
// the reply still open, as when a broken stream is retried.
const (
	PieceClose Piece = ""
	// PieceDiscard is ASCII CAN, "cancel", which no model writes.
	PieceDiscard Piece = "\x18"
)

// Source ports hand out a fresh channel per call: consumers sharing one would steal from each other.
// SecondaryTextOut exists because a struct satisfies an interface once; split a block needing a third.
type (
	AudioOut         interface{ AudioOutPort() <-chan AudioChunk }
	TextOut          interface{ TextOutPort() <-chan string }
	SecondaryTextOut interface{ SecondaryTextOutPort() <-chan string }
	ImageOut         interface{ ImageOutPort() <-chan ImageData }
	PropsOut         interface{ PropsOutPort() <-chan PropMap }
	StreamOut        interface{ StreamOutPort() <-chan Piece }
)

// Phase is what a block is doing; a block reports its initial phase, then only changes.
type Phase string

const (
	PhaseReady   Phase = "ready"
	PhaseWorking Phase = "working"
	// PhaseFailed means the game cannot continue; Reason says why.
	PhaseFailed Phase = "failed"
)

// Reasons a block failed, named as the session's problems are.
const (
	ReasonKeyRejected = "key_rejected"
	ReasonUnreachable = "unreachable"
	ReasonFailed      = "failed"
)

// ReasonTurnFailed comes with PhaseReady: the block dropped the turn it was
// working on, and the game goes on.
const ReasonTurnFailed = "turn_failed"

// State is a block reporting on itself, named so a gate can tell reporters apart.
type State struct {
	Node  string `json:"node"`
	Phase Phase  `json:"phase"`
	// Reason and Message are set with PhaseFailed, and with PhaseReady when a
	// turn was dropped.
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type StateOut interface{ StateOutPort() <-chan State }

// Usage is a block's cumulative spend for the session: totals, so a dropped event costs
// accuracy only for an instant and a late subscriber sees the whole session.
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
	AudioIn  interface{ AudioInPort() chan<- AudioChunk }
	TextIn   interface{ TextInPort() chan<- string }
	ImageIn  interface{ ImageInPort() chan<- ImageData }
	PropsIn  interface{ PropsInPort() chan<- PropMap }
	StreamIn interface{ StreamInPort() chan<- Piece }
)

type Kind int

const (
	KindAudio Kind = iota
	KindText
	KindImage
	KindProps
	KindState
	KindStream
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
	case KindStream:
		return "stream"
	}
	return "unknown"
}

// Sink is an output block whose arrivals become one stream on the session's event stream.
type Sink interface {
	Named
	Stream() string
	Arrivals() chan string
}

// InputMode is how a player supplies input, which picks the client's control (mute toggle vs hold button).
type InputMode string

const (
	InputText            InputMode = "text"
	InputAudioFullDuplex InputMode = "audio-full-duplex"
	InputAudioPushToTalk InputMode = "audio-push-to-talk"
)

// Imagery says how many pictures a genre makes, which decides where a client places them.
type Imagery string

const (
	ImageryNone Imagery = ""
	// ImageryStanding is the character being spoken to, held in view for the whole session.
	ImageryStanding Imagery = "standing"
	ImageryPerTurn  Imagery = "per-turn"
)

// Switchable is a block whose work a session toggle turns off and on during play.
type Switchable interface{ SetActive(on bool) }

// Named lets a block report itself in graph errors and traces; otherwise its Go type name is used.
type Named interface{ NodeName() string }

// RequiresInputs lists the kinds a block cannot run without, for graph validation.
type RequiresInputs interface{ RequiredInputs() []Kind }

// RequiresOutputs lists the kinds something must consume, so a character nobody can hear fails validation.
type RequiresOutputs interface{ RequiredOutputs() []Kind }

// Starter is implemented by blocks that run their own loop.
type Starter interface{ Start(ctx context.Context) }

// Resumable blocks hold opaque provider-side state that must survive a restart.
// Both methods run while the block's own loop keeps going, so guard what they touch.
type Resumable interface {
	ExportState() string
	RestoreState(string)
}

// Gatekeeper waits for each feeding block to go working then ready before play begins.
// A gating block must complete that cycle even when it has nothing to do, or the gate never opens.
type Gatekeeper interface {
	ExpectReporters(n int)
}

// Releasable is a node held until released; with no release edge it is never held, so it cannot deadlock.
type Releasable interface {
	ExpectReleases(n int)
}
