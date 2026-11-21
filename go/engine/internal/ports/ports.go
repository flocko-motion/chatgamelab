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

// The stream's markers. A reply ends with PieceClose; PieceDiscard drops the
// reply still open, as when a broken stream is retried; a piece starting with
// PieceReplace replaces the open reply's text with the rest of the piece, as
// when a reply is cleaned up after it streamed.
const (
	PieceClose Piece = ""
	// PieceDiscard is ASCII CAN, "cancel", which no model writes.
	PieceDiscard Piece = "\x18"
	// PieceReplace is ASCII SUB, "substitute", which no model writes.
	PieceReplace Piece = "\x1a"
)

// AudioDiscard is the audio stream's one marker, an empty chunk: it drops the
// spoken reply still playing and queued, as when a broken stream is retried.
var AudioDiscard = AudioChunk{}

// Discard reports whether the chunk is AudioDiscard.
func (c AudioChunk) Discard() bool { return len(c) == 0 }

// Replacement is the piece that replaces the open reply's text with text.
func Replacement(text string) Piece { return PieceReplace + Piece(text) }

// Source ports hand out a fresh channel per call: consumers sharing one would steal from each other.
// SecondaryTextOut exists because a struct satisfies an interface once; split a block needing a third.
type (
	AudioOut         interface{ AudioOutPort() <-chan AudioChunk }
	TextOut          interface{ TextOutPort() <-chan string }
	SecondaryTextOut interface{ SecondaryTextOutPort() <-chan string }
	ImageOut         interface{ ImageOutPort() <-chan ImageData }
	PropsOut         interface{ PropsOutPort() <-chan PropMap }
	StreamOut        interface{ StreamOutPort() <-chan Piece }
	// FailureOut carries a block's call that failed for good; its normal
	// outputs then carry nothing.
	FailureOut interface{ FailureOutPort() <-chan Failure }
)

// Phase is what a block is doing; a block reports its initial phase, then only changes.
type Phase string

const (
	PhaseReady   Phase = "ready"
	PhaseWorking Phase = "working"
)

// Reasons a block's call failed, named as the session's problems are. A
// failure is the block's: what happens next is its failure edge's.
const (
	ReasonKeyRejected = "key_rejected"
	ReasonUnreachable = "unreachable"
	// ReasonBadReply: every attempt answered something the block could not use.
	ReasonBadReply = "bad_reply"
	// ReasonNothingHeard: the model heard no words in the player's speech.
	ReasonNothingHeard = "nothing_heard"
	ReasonFailed       = "failed"
)

// State is a block reporting on itself, named so a gate can tell reporters apart.
type State struct {
	Node  string `json:"node"`
	Phase Phase  `json:"phase"`
	// Reason and Message are set with the PhaseReady that follows a call that
	// failed for good: the provider's reason, kept for the graph view.
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// Failure is a block's call that failed for good: the input it was given,
// and the provider's reason, named as State's.
type Failure struct {
	Node    string `json:"node"`
	Input   string `json:"input,omitempty"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// Failed says the report follows a call that failed for good.
func (s State) Failed() bool { return s.Reason != "" }

type StateOut interface{ StateOutPort() <-chan State }

// Usage is a block's cumulative spend for the session: totals, so a dropped event costs
// accuracy only for an instant and a late subscriber sees the whole session.
type Usage struct {
	Node              string `json:"node"`
	Model             string `json:"model"`
	InputTokens       int64  `json:"inputTokens,omitempty"`
	CachedInputTokens int64  `json:"cachedInputTokens,omitempty"`
	OutputTokens      int64  `json:"outputTokens,omitempty"`
	// The audio tokens are a speaking model's audio, billed apart from text.
	AudioInputTokens       int64   `json:"audioInputTokens,omitempty"`
	CachedAudioInputTokens int64   `json:"cachedAudioInputTokens,omitempty"`
	AudioOutputTokens      int64   `json:"audioOutputTokens,omitempty"`
	AudioSeconds           float64 `json:"audioSeconds,omitempty"`
	Images                 int64   `json:"images,omitempty"`
}

type UsageOut interface{ UsageOutPort() <-chan Usage }
type StateIn interface{ StateInPort() chan<- State }

type (
	AudioIn  interface{ AudioInPort() chan<- AudioChunk }
	TextIn   interface{ TextInPort() chan<- string }
	ImageIn  interface{ ImageInPort() chan<- ImageData }
	PropsIn  interface{ PropsInPort() chan<- PropMap }
	StreamIn interface{ StreamInPort() chan<- Piece }
	// SecondaryTextIn is a block's second text input, apart from its first.
	SecondaryTextIn interface{ SecondaryTextInPort() chan<- string }
	// OpeningIn takes the cue that opens the game, apart from the turns.
	OpeningIn interface{ OpeningInPort() chan<- string }
	// FailureIn takes failures whole, as a sink that reports them does.
	FailureIn interface{ FailureInPort() chan<- Failure }
)

type Kind int

const (
	KindAudio Kind = iota
	KindText
	KindImage
	KindProps
	KindState
	KindStream
	// KindFailure is a failure edge, drawn apart from the normal flow.
	KindFailure
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
	case KindFailure:
		return "failure"
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

// Origin is a block that starts its work itself, as a source does, though
// answers flow back into it; the completeness check walks from it.
type Origin interface{ Origin() }

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
