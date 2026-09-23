// Package adapters is the seam between a block and whatever actually executes
// it. A block holds an interface from this package; which model, platform and
// key it resolves to is settled in the SessionSpec before the graph is built.
package adapters

import "context"

// EventKind distinguishes what arrived on a live connection. Audio and
// Transcript arrive interleaved and continuously; the model's transcript is its
// own output, so it is exact rather than a guess at what it said.
type EventKind int

const (
	EventAudio EventKind = iota
	EventTranscript
	EventTurnComplete
	EventError
)

func (k EventKind) String() string {
	switch k {
	case EventAudio:
		return "audio"
	case EventTranscript:
		return "transcript"
	case EventTurnComplete:
		return "turn-complete"
	case EventError:
		return "error"
	}
	return "unknown"
}

type LiveEvent struct {
	Kind  EventKind
	Audio []byte
	Text  string
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
	// Instruct replaces the session's standing instructions mid-conversation,
	// which is how the observer steers a drifting character.
	Instruct(text string) error
	Events() <-chan LiveEvent
	Close() error
}

type Live interface {
	Open(ctx context.Context, cfg LiveConfig) (LiveConn, error)
}

// Image generates one picture from a prompt. Single-shot by nature: nothing
// about an image is continued.
type Image interface {
	Generate(ctx context.Context, prompt string) ([]byte, error)
}

// KeyFunc yields the API key at the moment it is needed, rather than holding a
// copy. An adapter calls it per connection or per request, so nothing here ever
// stores a secret.
type KeyFunc func(ctx context.Context) (string, error)

// Tool is a single-shot text transformation. The role is config: the same
// adapter serves the observer's classifier, a rephrase, or a translation.
type Tool interface {
	Query(ctx context.Context, system, user string) (string, error)
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
