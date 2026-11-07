// Package adapters is the seam between a block and whatever executes it; the
// SessionSpec settles model, platform and key before the graph is built.
package adapters

import "context"

// EventKind is the mechanism by which something arrived; who spoke is Speaker.
type EventKind int

const (
	// EventStarted says the session is running and will accept commands.
	EventStarted EventKind = iota
	// EventConnect carries what the browser needs to open its own audio connection.
	EventConnect
	EventAudio
	EventText
	EventUsage
	EventClosed
	EventError
)

func (k EventKind) String() string {
	switch k {
	case EventStarted:
		return "started"
	case EventConnect:
		return "connect"
	case EventAudio:
		return "audio"
	case EventText:
		return "text"
	case EventUsage:
		return "usage"
	case EventClosed:
		return "closed"
	case EventError:
		return "error"
	}
	return "unknown"
}

// Speaker says whose half of the conversation a transcript fragment belongs to.
type Speaker int

const (
	SpeakerCharacter Speaker = iota
	SpeakerPlayer
)

// LiveEvent is one thing that happened on a live conversation. StartMs and EndMs
// are the API's only ordering, so reconstruction reads them, never arrival order.
type LiveEvent struct {
	Kind    EventKind
	Speaker Speaker
	Audio   []byte
	Text    string
	StartMs int64
	EndMs   int64
	Usage   Usage
	// CloseReason "content" is the provider's safety filter, distinct from a disconnect.
	CloseReason string
	Err         error
}

// LiveConfig keeps Scenario and Guardrail apart to the wire, so designer text
// cannot reach the platform's constraint.
type LiveConfig struct {
	Voice     string
	Scenario  string
	Guardrail string

	// Offer is the browser's SDP offer; answering it creates, and bills, the session.
	Offer string

	// ResumeFrom names the conversation to fork; empty starts a fresh one.
	ResumeFrom string

	// History seeds a fresh session when no fork is possible: both speakers, oldest
	// first, from the transcript the engine keeps in process memory only.
	History []Utterance
}

// Utterance is one thing somebody said, for restoring a conversation.
type Utterance struct {
	Speaker Speaker
	Text    string
}

// LiveConn is the engine's control connection to one conversation; audio bypasses it.
type LiveConn interface {
	// ID is what a resume quotes, so whoever may resume must keep it.
	ID() string
	// Answer is the SDP answer the browser needs to complete its connection.
	Answer() string
	// Instruct appends, so re-injecting the guardrail keeps the character.
	Instruct(ctx context.Context, text string) error
	Events() <-chan LiveEvent
	Close() error
}

// Live creates conversations.
type Live interface {
	Open(ctx context.Context, cfg LiveConfig) (LiveConn, error)
}

// Usage is what a call cost, in the units the provider bills.
type Usage struct {
	Model string `json:"model"`
	// AudioSeconds counts time the session is open: a silent minute costs a minute.
	InputTokens       int64   `json:"inputTokens,omitempty"`
	CachedInputTokens int64   `json:"cachedInputTokens,omitempty"`
	OutputTokens      int64   `json:"outputTokens,omitempty"`
	AudioSeconds      float64 `json:"audioSeconds,omitempty"`
	Images            int64   `json:"images,omitempty"`
}

// Add folds in another call's usage, so a block reports its running total.
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

// Price is supplied by the caller: providers publish no prices through their APIs.
type Price struct {
	InputPerMTok       float64
	CachedInputPerMTok float64
	OutputPerMTok      float64
	AudioPerMinute     float64
	PerImage           float64
}

// Cost is an estimate; only the provider's bill is authoritative.
func (p Price) Cost(u Usage) float64 {
	const perMillion = 1_000_000
	return float64(u.InputTokens)/perMillion*p.InputPerMTok +
		float64(u.CachedInputTokens)/perMillion*p.CachedInputPerMTok +
		float64(u.OutputTokens)/perMillion*p.OutputPerMTok +
		u.AudioSeconds/60*p.AudioPerMinute +
		float64(u.Images)*p.PerImage
}

// ImageRequest carries only the prompt; size and quality come with the tier.
type ImageRequest struct {
	Prompt string
}

// Image generates one picture.
type Image interface {
	Generate(ctx context.Context, req ImageRequest) ([]byte, Usage, error)
}

// KeyFunc yields the key per connection or request, so no adapter stores a secret.
type KeyFunc func(ctx context.Context) (string, error)

// Tool is a single-shot text transformation; the role is config.
type Tool interface {
	Query(ctx context.Context, system, user string) (string, Usage, error)
}

// Tier is the session's quality setting, resolved per platform into a model per role.
type Tier string

const (
	TierEconomy  Tier = "economy"
	TierBalanced Tier = "balanced"
	TierPremium  Tier = "premium"
	TierMax      Tier = "max"
)

// ModelSet is one tier resolved into a model per role, named by mechanism; an
// empty field means the role is unavailable at that tier.
type ModelSet struct {
	Live     string
	Tool     string
	Threaded string
	Image    string
	Audio    string
	// ImageQuality and ImageSize are priced with the image model, so they ride with it.
	ImageQuality string
	ImageSize    string
}
