package engine

import (
	"time"

	"engine/internal/ports"
)

// A game can be over: final, for a reason, unlike a problem, which the
// session gets past by itself or once someone fixes it. A game that is over
// is read back as it was played, its last line saying why it ended, and is
// never played on.

// OverReason says why a game is over, as a code the playback-engine tells the
// player in their own language.
type OverReason string

const (
	// OverContentPolicy: the AI provider's safety filter ended the game. It is
	// the provider's decision, and the engine keeps to it.
	OverContentPolicy OverReason = "content_policy"
)

// OverReasons is every reason, the source of the generated TypeScript union.
func OverReasons() []OverReason { return []OverReason{OverContentPolicy} }

// Over is a game's end: why, and when.
type Over struct {
	Reason OverReason `json:"reason"`
	At     time.Time  `json:"at"`
}

// StreamOver carries the reason a game is over; it is the last line of the history.
const StreamOver = "over"

// end makes the game over, once: the reason goes into the history as its last
// line and onto the stream, and the state is saved with it. Play is refused
// from then on.
func (s *Session) end(reason OverReason) {
	s.mu.Lock()
	if s.over != nil {
		s.mu.Unlock()
		return
	}
	s.over = &Over{Reason: reason, At: time.Now()}
	s.mu.Unlock()
	s.journal.Note(JournalSession, "over", string(reason))
	ev := SessionEvent{Stream: StreamOver, Value: string(reason)}
	s.log(ev)
	s.hub.publish(ev)
	s.checkpoint()
}

// overNow is the game's end, or nil while it goes on.
func (s *Session) overNow() *Over {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.over == nil {
		return nil
	}
	over := *s.over
	return &over
}

// forwardStops ends the game when a block says it must stop, by the reason it gives.
func (s *Session) forwardStops(stops <-chan ports.State) {
	if stops == nil {
		return
	}
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case st, alive := <-stops:
				if !alive {
					return
				}
				s.end(OverReason(st.Reason))
			}
		}
	}()
}
