package ports

import (
	"strings"
	"sync"
	"time"
)

// JournalCapacity is how many entries a session's journal keeps; older ones fall off.
const JournalCapacity = 2000

// journalText is the longest detail an entry keeps; a picture or a long reply is cut.
const journalText = 400

// JournalEntry is one thing that happened in a session, as the engine saw it.
type JournalEntry struct {
	Time time.Time `json:"time"`
	// Source is who noted it: a block's name, "session", or "client".
	Source string `json:"source"`
	Level  Level  `json:"level"`
	Kind   string `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

// Level says whether an entry is a problem: a warning the game carried on
// after, or an error that lost something.
type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

// Journal is a session's server-side record of what happened, kept in memory
// and cut to its capacity, so a stuck session can be read back however often
// its page reloads. A nil Journal notes nothing.
type Journal struct {
	mu       sync.Mutex
	entries  []JournalEntry
	next     int
	capacity int
	clock    func() time.Time
	// listen hears every entry as it is noted, outside the lock.
	listen func(JournalEntry)
}

func NewJournal(capacity int, clock func() time.Time) *Journal {
	if clock == nil {
		clock = time.Now
	}
	return &Journal{capacity: capacity, clock: clock}
}

// Listen hands every later entry to fn as it is noted; set once, before the session runs.
func (j *Journal) Listen(fn func(JournalEntry)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.listen = fn
}

// Note records something that happened.
func (j *Journal) Note(source, kind, detail string) { j.add(source, LevelInfo, kind, detail) }

// Warn records a problem the game carried on after.
func (j *Journal) Warn(source, kind, detail string) { j.add(source, LevelWarning, kind, detail) }

// Fail records a problem that lost something: a call, a conversation, a picture.
func (j *Journal) Fail(source, kind, detail string) { j.add(source, LevelError, kind, detail) }

func (j *Journal) add(source string, level Level, kind, detail string) {
	if j == nil {
		return
	}
	if len(detail) > journalText {
		detail = strings.ToValidUTF8(detail[:journalText], "") + "…"
	}
	j.mu.Lock()
	e := JournalEntry{Time: j.clock().UTC(), Source: source, Level: level, Kind: kind, Detail: detail}
	if len(j.entries) < j.capacity {
		j.entries = append(j.entries, e)
	} else {
		j.entries[j.next] = e
		j.next = (j.next + 1) % j.capacity
	}
	listen := j.listen
	j.mu.Unlock()
	if listen != nil {
		listen(e)
	}
}

// Restore takes back entries noted before, such as a stored session's,
// oldest first, ahead of anything noted since; nobody listening hears them.
func (j *Journal) Restore(entries []JournalEntry) {
	if j == nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	all := append(append([]JournalEntry(nil), entries...), j.ordered()...)
	if len(all) > j.capacity {
		all = all[len(all)-j.capacity:]
	}
	j.entries, j.next = all, 0
}

// ordered is the entries oldest first; the caller holds mu.
func (j *Journal) ordered() []JournalEntry {
	out := make([]JournalEntry, 0, len(j.entries))
	out = append(out, j.entries[j.next:]...)
	return append(out, j.entries[:j.next]...)
}

// Entries are the entries kept, oldest first.
func (j *Journal) Entries() []JournalEntry {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.ordered()
}

// Journaling is a node that notes into the session's journal; the graph hands
// it the journal before it starts.
type Journaling interface {
	UseJournal(*Journal)
}
