package ports

import (
	"strings"
	"testing"
	"time"
)

func TestJournalKeepsTheLatestEntriesInOrder(t *testing.T) {
	j := NewJournal(3, nil)
	for _, kind := range []string{"a", "b", "c", "d", "e"} {
		j.Note("x", kind, "")
	}
	var kinds []string
	for _, e := range j.Entries() {
		kinds = append(kinds, e.Kind)
	}
	if got := strings.Join(kinds, ""); got != "cde" {
		t.Errorf("kept %q, want the latest three oldest first, cde", got)
	}
}

func TestJournalSaysEachEntrysLevelAndHandsItOn(t *testing.T) {
	at := time.Date(2026, 11, 23, 21, 44, 0, 0, time.UTC)
	j := NewJournal(10, func() time.Time { return at })
	var heard []JournalEntry
	j.Listen(func(e JournalEntry) { heard = append(heard, e) })

	j.Note("live-session", "opened", "conversation 1")
	j.Warn("live-session", "closed", "remote_hangup")
	j.Fail("image", "call-failed", "unreachable")

	want := []Level{LevelInfo, LevelWarning, LevelError}
	if len(heard) != len(want) {
		t.Fatalf("the listener heard %d entries, want %d", len(heard), len(want))
	}
	for i, e := range j.Entries() {
		if e.Level != want[i] || heard[i] != e {
			t.Errorf("entry %d: level %s, heard %+v; want level %s, heard as kept", i, e.Level, heard[i], want[i])
		}
		if !e.Time.Equal(at) {
			t.Errorf("entry %d at %v, want the clock's %v", i, e.Time, at)
		}
	}
}

func TestJournalCutsALongDetail(t *testing.T) {
	j := NewJournal(1, nil)
	j.Note("session", "image", strings.Repeat("ä", journalText))
	detail := j.Entries()[0].Detail
	if len(detail) > journalText+len("…") || !strings.HasSuffix(detail, "…") {
		t.Errorf("a long detail is kept at %d bytes; want it cut to %d and marked", len(detail), journalText)
	}
	if !strings.HasPrefix(detail, "ää") || strings.ContainsRune(detail, '�') {
		t.Error("the cut broke a character in two")
	}
}

func TestNilJournalNotesNothing(t *testing.T) {
	var j *Journal
	j.Note("x", "y", "z")
	if j.Entries() != nil {
		t.Error("a nil journal holds entries")
	}
}
