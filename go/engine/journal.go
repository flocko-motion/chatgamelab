package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engine/internal/ports"
)

// JournalEntry is one thing that happened in a session, as the engine saw it.
type JournalEntry = ports.JournalEntry

// The journal's sources beside the blocks, which note under their own names.
const (
	JournalSession = "session"
	JournalClient  = "client"
)

// Journal is what happened in the session so far, oldest first: the engine's
// own record, kept in memory, which a reloading page reads back whole.
func (s *Session) Journal() []JournalEntry { return s.journal.Entries() }

// Note records something a caller outside the blocks saw, such as a client attaching.
func (s *Session) Note(source, kind, detail string) { s.journal.Note(source, kind, detail) }

// Warn records a problem a caller outside the blocks saw.
func (s *Session) Warn(source, kind, detail string) { s.journal.Warn(source, kind, detail) }

// journalBacklog is how many entries wait to be stored before new ones are
// dropped, which only a store far slower than play could reach.
const journalBacklog = 4096

// openJournal starts the session's journal. Each entry is stored with the
// session, beside its history, so a session reported broken can be read
// back whole, across restarts, and handed to every page as a "journal" event.
func (s *Session) openJournal() {
	s.journal = ports.NewJournal(ports.JournalCapacity, nil)
	s.journalKeep = make(chan SessionEvent, journalBacklog)
	s.journalKept = make(chan struct{})
	s.journal.Listen(func(e JournalEntry) {
		blob, err := json.Marshal(e)
		if err != nil {
			return
		}
		ev := SessionEvent{Stream: "journal", Value: string(blob)}
		select {
		case s.journalKeep <- ev:
		default:
		}
		s.hub.publish(ev)
	})
}

// keepJournal stores the journal's entries in order until the session ends,
// and those still waiting then.
func (s *Session) keepJournal() {
	defer close(s.journalKept)
	save := func(ev SessionEvent) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(s.ctx), saveTimeout)
		defer cancel()
		s.saveEvent(ctx, ev)
	}
	for {
		select {
		case ev := <-s.journalKeep:
			save(ev)
		case <-s.ctx.Done():
			for {
				select {
				case ev := <-s.journalKeep:
					save(ev)
				default:
					return
				}
			}
		}
	}
}

// restoreJournal takes back the journal a stored session kept.
func (s *Session) restoreJournal(history []SessionEvent) {
	var entries []JournalEntry
	for _, e := range history {
		if e.Stream != "journal" {
			continue
		}
		var entry JournalEntry
		if json.Unmarshal([]byte(e.Value), &entry) == nil {
			entries = append(entries, entry)
		}
	}
	s.journal.Restore(entries)
}

// sinkJournal notes what one sink delivers: text whole once its utterance
// ends, audio not at all, a picture by its size, anything else as it is.
type sinkJournal struct {
	s      *Session
	stream string
	open   strings.Builder
}

func (j *sinkJournal) note(value string) {
	switch j.stream {
	case "audio":
	case "text", "text-b", "said":
		if value != "" {
			j.open.WriteString(value)
			return
		}
		if j.open.Len() > 0 {
			j.s.journal.Note(JournalSession, j.stream, j.open.String())
			j.open.Reset()
		}
	case "image":
		j.s.journal.Note(JournalSession, "image", fmt.Sprintf("a picture of %d bytes", len(value)))
	case "error":
		j.s.journal.Warn(JournalSession, "error", value)
	default:
		j.s.journal.Note(JournalSession, j.stream, value)
	}
}
