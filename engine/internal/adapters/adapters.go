// Package adapters is the seam between a block and whatever actually executes
// it. A block holds an interface from this package; which model, platform and
// key it resolves to is settled in the SessionSpec before the graph is built.
package adapters

import "context"

// EventKind distinguishes what arrived on a live connection.
//
// Named for the mechanism, not the use case: a live adapter emits text beside
// audio, and which speaker it belongs to is a separate field because GPT-Live
// transcribes both halves of the conversation.
type EventKind int

const (
	// EventStarted says the session is running and will accept commands.
	EventStarted EventKind = iota
	// EventConnect carries what the browser needs to open its own audio
	// connection. The engine never holds that connection.
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
// GPT-Live transcribes both, and they are not interchangeable: one is what the
// character said, the other what the player said.
type Speaker int

const (
	SpeakerCharacter Speaker = iota
	SpeakerPlayer
)

// LiveEvent is one thing that happened on a live conversation.
//
// StartMs and EndMs place a transcript fragment on the session timeline. They
// are the only ordering the API provides — there is no turn bracket and no
// event marking the end of an utterance — so anything reconstructing a
// conversation reads these rather than arrival order.
type LiveEvent struct {
	Kind    EventKind
	Speaker Speaker
	Audio   []byte
	Text    string
	StartMs int64
	EndMs   int64
	Usage   Usage
	// CloseReason says why a session ended. "content" means the provider's own
	// safety filter stopped it, which is youth protection firing inside the
	// model and deserves to be seen as itself.
	CloseReason string
	Err         error
}

// LiveConfig is what a live conversation needs at creation.
//
// Scenario and Guardrail stay separate all the way to the wire: the scenario
// becomes the model's standing instructions, the guardrail a developer message.
// Collapsing them into one string would make their separation a matter of
// discipline, where keeping them in different fields makes it structural — no
// scenario text, whatever it contains, can reach the field the guardrail holds.
type LiveConfig struct {
	Voice     string
	Scenario  string
	Guardrail string

	// Offer is the browser's SDP offer. Creating the session is the exchange
	// that answers it, which is why a live conversation cannot be opened until
	// a player has actually arrived.
	Offer string

	// ResumeFrom continues a conversation that was closed, named by the id of
	// the one it continues. A conversation costs money for every second it
	// stays open, so an idle one is closed rather than left running — and
	// picking it back up must not cost the character its memory.
	//
	// Empty starts a fresh conversation.
	ResumeFrom string

	// History is what was said, oldest first, for continuing a conversation
	// whose recording the provider does not have. It is the fallback path and
	// the one that has to work: a recording finalises only on a graceful close,
	// which a dropped connection is not.
	History []Utterance
}

// Utterance is one thing somebody said, for restoring a conversation.
type Utterance struct {
	Speaker Speaker
	Text    string
}

// LiveConn is one open conversation. The engine holds a control connection to
// it; the audio runs between the player's browser and the provider.
type LiveConn interface {
	// ID names this conversation to the provider. Continuing it later means
	// naming it, so whoever may want to resume has to keep this.
	ID() string
	// Answer is the SDP answer the browser needs to complete its connection.
	Answer() string
	// Instruct appends to the session's standing instructions, which is how the
	// observer steers a drifting character and how the gate opens the game. It
	// appends rather than replaces, so re-injecting the guardrail never
	// discards the character along with it.
	Instruct(ctx context.Context, text string) error
	Events() <-chan LiveEvent
	Close() error
}

// Live creates conversations. Open brokers the browser's offer for an answer
// and attaches the engine's own control connection.
type Live interface {
	Open(ctx context.Context, cfg LiveConfig) (LiveConn, error)
}

// Usage is what a call cost, in the units the provider bills. Tokens and
// seconds both appear because they are billed differently: text models charge
// per token, and a live conversation charges for the time it stays open.
//
// Cached input is separate because it is priced separately — the game flow hits
// the prompt cache from the second turn onwards.
type Usage struct {
	Model string `json:"model"`
	// Three units, because providers bill in three ways: text per token, live
	// conversation per second of session duration, and pictures per picture.
	// AudioSeconds therefore counts an open session rather than speech: a
	// silent minute costs a minute.
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

// ImageRequest is one picture to make. Only the prompt varies per call: what
// size and quality a session buys is settled with its tier, before any block
// asks for anything.
type ImageRequest struct {
	Prompt string
}

// Image generates one picture. Single-shot by nature: nothing about an image is
// continued.
type Image interface {
	Generate(ctx context.Context, req ImageRequest) ([]byte, Usage, error)
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
	// ImageQuality and ImageSize ride with the image model because they are
	// priced with it: a tier buys its place on the ladder with how good the
	// picture is rather than by dropping the feature.
	ImageQuality string
	ImageSize    string
}
