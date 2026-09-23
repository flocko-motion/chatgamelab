// Package adapters is the seam between a block and whatever actually executes
// it. A block holds an interface from this package; which model, platform and
// key it resolves to is settled in the SessionSpec before the graph is built.
package adapters

import "context"

// EventKind distinguishes what arrived on a live connection. Audio and text
// arrive interleaved and continuously, and the text is the model's own output
// rather than a guess at what it said.
//
// Named for the mechanism, not the use case: a live adapter emits text beside
// audio, which is a transcript only when the model is narrating its own speech.
type EventKind int

const (
	EventAudio EventKind = iota
	EventText
	EventTurnComplete
	EventUsage
	EventError
)

func (k EventKind) String() string {
	switch k {
	case EventAudio:
		return "audio"
	case EventText:
		return "text"
	case EventTurnComplete:
		return "turn-complete"
	case EventUsage:
		return "usage"
	case EventError:
		return "error"
	}
	return "unknown"
}

type LiveEvent struct {
	Kind  EventKind
	Audio []byte
	Text  string
	Usage Usage
	Err   error
}

// LiveConfig is what a live connection needs at open time. Instructions carry
// the character and the platform's resolved guardrail; both can be replaced
// mid-session through Instruct.
type LiveConfig struct {
	Voice        string
	Instructions string
}

// LiveConn is one open speech-to-speech conversation. It is full-duplex:
// Send and Events run concurrently for the connection's lifetime.
type LiveConn interface {
	Send(audio []byte) error
	// SendText submits typed input. A live session takes both, which matters
	// for testing without a microphone and for anyone who would rather type.
	SendText(text string) error
	// Instruct replaces the session's standing instructions mid-conversation,
	// which is how the observer steers a drifting character.
	Instruct(text string) error
	Events() <-chan LiveEvent
	Close() error
}

type Live interface {
	Open(ctx context.Context, cfg LiveConfig) (LiveConn, error)
}

// Usage is what a call cost, in the units the provider bills. Tokens and
// seconds both appear because they are billed differently: text models charge
// per token, and a live audio session charges per minute of audio.
//
// Cached input is separate because it is priced separately — the game flow hits
// the prompt cache from the second turn onwards.
type Usage struct {
	Model string `json:"model"`
	// Three units, because providers bill in three ways: text per token, live
	// audio per minute, and pictures per picture.
	InputTokens       int64   `json:"inputTokens,omitempty"`
	CachedInputTokens int64   `json:"cachedInputTokens,omitempty"`
	OutputTokens      int64   `json:"outputTokens,omitempty"`
	AudioSeconds      float64 `json:"audioSeconds,omitempty"`
	Images            int64   `json:"images,omitempty"`
}

// Add accumulates another call's usage. Totals are kept per block, so a block
// reports what it has spent so far rather than a delta that a dropped event
// would lose.
func (u *Usage) Add(other Usage) {
	if other.Model != "" {
		u.Model = other.Model
	}
	u.InputTokens += other.InputTokens
	u.CachedInputTokens += other.CachedInputTokens
	u.OutputTokens += other.OutputTokens
	u.AudioSeconds += other.AudioSeconds
	u.Images += other.Images
}

// Price is what a model costs, in the units it is billed in. Supplied by the
// caller rather than discovered, because providers do not publish prices through
// their APIs.
type Price struct {
	InputPerMTok       float64
	CachedInputPerMTok float64
	OutputPerMTok      float64
	AudioPerMinute     float64
	PerImage           float64
}

// Cost estimates what this usage was worth. It is an estimate: the prices are
// hand-maintained and the provider is the only authority on a bill.
func (p Price) Cost(u Usage) float64 {
	const perMillion = 1_000_000
	return float64(u.InputTokens)/perMillion*p.InputPerMTok +
		float64(u.CachedInputTokens)/perMillion*p.CachedInputPerMTok +
		float64(u.OutputTokens)/perMillion*p.OutputPerMTok +
		u.AudioSeconds/60*p.AudioPerMinute +
		float64(u.Images)*p.PerImage
}

// Image generates one picture from a prompt. Single-shot by nature: nothing
// about an image is continued.
type Image interface {
	Generate(ctx context.Context, prompt string) ([]byte, Usage, error)
}

// KeyFunc yields the API key at the moment it is needed, rather than holding a
// copy. An adapter calls it per connection or per request, so nothing here ever
// stores a secret.
type KeyFunc func(ctx context.Context) (string, error)

// Tool is a single-shot text transformation. The role is config: the same
// adapter serves the observer's classifier, a rephrase, or a translation.
type Tool interface {
	Query(ctx context.Context, system, user string) (string, Usage, error)
}

// Tier is the session's quality setting. It resolves, per platform, into a
// concrete model for each role — the same ladder v1 documents, where every rung
// asks the same work of its model and buys its price with quality rather than
// by dropping features.
type Tier string

const (
	TierEconomy  Tier = "economy"
	TierBalanced Tier = "balanced"
	TierPremium  Tier = "premium"
	TierMax      Tier = "max"
)

// ModelSet is one tier resolved into model names, one per adapter role. An
// empty field means the role is unavailable at that tier — economy generates no
// images, and speech output is a top-tier feature.
// Roles are named by mechanism, like the blocks that use them: Tool is
// single-shot, Threaded is one continuing conversation, Live is full duplex.
// What a genre calls a role — Adventure's "plot", say — stays in the genre.
type ModelSet struct {
	Live     string
	Tool     string
	Threaded string
	Image    string
	Audio    string
}
