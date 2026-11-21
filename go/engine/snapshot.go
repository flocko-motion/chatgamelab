package engine

import (
	"strings"
	"sync"

	"engine/internal/ports"
)

// TextDiscard on the text stream drops the open utterance; an empty value closes it.
const TextDiscard = string(ports.PieceDiscard)

// Snapshot is everything a client needs to rebuild its view without having
// watched the session happen — which is what a page reload is.
//
// It carries current values rather than a log: the latest phase per block, the
// running cost, the status fields. History is separate, because the
// conversation is a sequence and these are not.
type Snapshot struct {
	// Started is false while the init stem is still running.
	Started bool `json:"started"`
	// Phases is what each block is doing, keyed by node name.
	Phases map[string]string `json:"phases"`
	// Props are the status values a genre tracks.
	Props map[string]string `json:"props"`
	Usage UsageReport       `json:"usage"`
	// Conversation is where a live genre's conversation stands: waiting to be
	// opened, running, or let go. Empty for a genre that holds none.
	//
	// It is here rather than on the stream because it is the one thing a
	// reloading page cannot be shown: the invitation to connect was sent while
	// that page did not exist, and a live connection cannot be replayed. A page
	// that is not told sits waiting for a moment that has already passed.
	Conversation string `json:"conversation,omitempty"`

	// Choosing is true while the session waits for the player's choice, so a
	// reloading page shows the dialog again.
	Choosing bool `json:"choosing"`
	// Settled is what the session runs on, once it has settled.
	Settled *Settled `json:"settled,omitempty"`
}

// recorder keeps what a reloading client will ask for. The engine holds it
// because the client cannot: a session outlives any one page.
type recorder struct {
	mu sync.Mutex

	phases map[string]string
	props  map[string]string

	// history is the conversation, in order. Audio and phase churn are left out
	// on purpose: they are how a session sounded and felt as it happened, not
	// what it was, and replaying thirty audio frames per utterance into a
	// reloaded page would be neither useful nor cheap.
	history []SessionEvent

	// open is the utterance still streaming, kept whole once closed.
	open strings.Builder
}

// historyStreams is what a reload needs to redraw the conversation.
var historyStreams = map[string]bool{
	"text":  true,
	"image": true,
	"props": true,
	"error": true,
	// Phase changes are in here for one reason: the transition back to ready is
	// what says an utterance ended, and without it a replayed conversation runs
	// together into a single paragraph.
	"state": true,
}

const historyLimit = 2000

func newRecorder() *recorder {
	return &recorder{phases: map[string]string{}, props: map[string]string{}}
}

// record keeps a history event and returns what it kept, to store in order.
// Text is kept as one event per finished utterance, then the close.
func (r *recorder) record(e SessionEvent) []SessionEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !historyStreams[e.Stream] {
		return nil
	}
	kept := []SessionEvent{e}
	if e.Stream == "text" {
		switch e.Value {
		case TextDiscard:
			r.open.Reset()
			return nil
		case "":
			if r.open.Len() == 0 {
				return nil
			}
			kept = []SessionEvent{{Stream: "text", Value: r.open.String()}, e}
			r.open.Reset()
		default:
			r.open.WriteString(e.Value)
			return nil
		}
	}
	r.history = append(r.history, kept...)
	r.trim()
	return kept
}

// restore takes a stored history back, with the status values its last
// props event carried.
func (r *recorder) restore(history []SessionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range history {
		if !historyStreams[e.Stream] {
			continue
		}
		r.history = append(r.history, e)
		if e.Stream != "props" {
			continue
		}
		if props := readProps(e.Value); props != nil {
			r.props = props
		}
	}
	r.trim()
}

// trim keeps the newest events. A conversation long enough to hit the limit
// has lost nothing a player would scroll back to. Called with mu held.
func (r *recorder) trim() {
	if len(r.history) > historyLimit {
		r.history = r.history[len(r.history)-historyLimit:]
	}
}

func (r *recorder) recordPhase(node, phase string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.phases[node] = phase
}

func (r *recorder) recordProps(props map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.props = props
}

func (r *recorder) snapshot(started bool, usage UsageReport, conversation string) Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	phases := make(map[string]string, len(r.phases))
	for node, phase := range r.phases {
		phases[node] = phase
	}
	props := make(map[string]string, len(r.props))
	for key, value := range r.props {
		props[key] = value
	}
	return Snapshot{
		Started:      started,
		Phases:       phases,
		Props:        props,
		Usage:        usage,
		Conversation: conversation,
	}
}

func (r *recorder) replay() []SessionEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Empty rather than nil: a session nobody has spoken in yet still has a
	// history, and it is an empty one. A nil slice encodes as JSON null, which
	// a client iterating what it was handed cannot read.
	return append(make([]SessionEvent, 0, len(r.history)), r.history...)
}
