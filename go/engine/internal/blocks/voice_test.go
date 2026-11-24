package blocks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/ports"
)

// runVoice runs the block once and answers the voice it put out, "" for none,
// and what it journalled.
func runVoice(t *testing.T, style string, maker adapters.Voices, store adapters.VoiceStore) (string, []ports.JournalEntry) {
	t.Helper()
	return runVoiceCapped(t, style, maker, store, 90)
}

// runVoiceCapped is runVoice with a cap on the voices of ours a key keeps.
func runVoiceCapped(t *testing.T, style string, maker adapters.Voices, store adapters.VoiceStore, capped int) (string, []ports.JournalEntry) {
	t.Helper()
	slot := &adapters.VoicesSlot{}
	slot.Set(maker, store)
	b := NewVoice("voice", style, slot)
	b.Wait, b.Poll, b.Cap = 2*time.Second, 10*time.Millisecond, capped
	b.Makes = true
	b.Builtin = []string{"marin", "stone"}
	b.Fallback = "marin"
	// The brief's two fields, as their wires bring them.
	b.SecondaryTextInPort() <- "stone"
	b.TextInPort() <- style + ", as a voice"
	journal := ports.NewJournal(100, nil)
	b.UseJournal(journal)
	out, state := b.TextOutPort(), b.StateOutPort()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	voice := ""
	ready := 0
	deadline := time.After(3 * time.Second)
	// Until both: the voice's output closed and the block reported ready again.
	for ready < 2 || out != nil {
		select {
		case v, ok := <-out:
			if ok {
				voice = v
			} else {
				out = nil
			}
		case s, ok := <-state:
			if !ok {
				state = nil
				ready = 2
				continue
			}
			if s.Phase == ports.PhaseReady {
				ready++
			}
		case <-deadline:
			t.Fatal("the voice block never reported ready")
		}
	}
	return voice, journal.Entries()
}

func kinds(entries []ports.JournalEntry) string {
	var k []string
	for _, e := range entries {
		k = append(k, e.Kind)
	}
	return strings.Join(k, ",")
}

// A voice is made once per voice style and key, and every later session
// reuses it.
func TestVoiceIsMadeOnceAndReused(t *testing.T) {
	maker, store := &mock.Voices{}, &adapters.MemoryVoices{}

	first, j1 := runVoice(t, "deep and slow", maker, store)
	if first == "" || !strings.Contains(kinds(j1), "created") {
		t.Fatalf("no voice made: %q, journal %s", first, kinds(j1))
	}
	second, j2 := runVoice(t, "deep and slow", maker, store)
	if second != first || !strings.Contains(kinds(j2), "reused") {
		t.Errorf("the second session got %q (%s), want the first's %q reused", second, kinds(j2), first)
	}
	if maker.Made.Load() != 1 {
		t.Errorf("made %d voices, want 1", maker.Made.Load())
	}

	// Another voice style is another voice.
	if other, _ := runVoice(t, "bright and quick", maker, store); other == first || other == "" {
		t.Errorf("another style got %q", other)
	}
}

// Two sessions starting at once make one voice: the second waits for it.
func TestVoiceStartedTwiceAtOnceIsMadeOnce(t *testing.T) {
	maker, store := &mock.Voices{Pace: 100 * time.Millisecond}, &adapters.MemoryVoices{}
	var wg sync.WaitGroup
	got := make([]string, 2)
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i], _ = runVoice(t, "deep and slow", maker, store)
		}(i)
	}
	wg.Wait()
	if got[0] == "" || got[0] != got[1] {
		t.Errorf("the two sessions speak in %q and %q, want one voice", got[0], got[1])
	}
	if maker.Made.Load() != 1 {
		t.Errorf("made %d voices, want 1", maker.Made.Load())
	}
}

// Without a voice style, a maker or a store, the built-in voice speaks; a
// voice that cannot be made leaves it too, and the claim goes, so the next
// session tries again.
func TestVoiceFallsBackToTheBuiltInVoice(t *testing.T) {
	if v, j := runVoice(t, "  ", &mock.Voices{}, &adapters.MemoryVoices{}); v != "stone" || !strings.Contains(kinds(j), "built-in") {
		t.Errorf("no style: %q, %s, want the built-in voice the brief proposed", v, kinds(j))
	}
	if v, j := runVoice(t, "deep", nil, nil); v != "stone" || !strings.Contains(kinds(j), "built-in") {
		t.Errorf("no maker: %q, %s, want the built-in voice the brief chose", v, kinds(j))
	}

	store := &adapters.MemoryVoices{}
	broken := &mock.Voices{Fail: errors.New("voices api: 400 Bad Request: refused")}
	v, j := runVoice(t, "deep", broken, store)
	if v != "stone" || !strings.Contains(kinds(j), "create-failed") {
		t.Errorf("a failed voice: %q, %s", v, kinds(j))
	}
	if again, _ := runVoice(t, "deep", &mock.Voices{}, store); again == "" {
		t.Error("the failed claim was kept, so no later session could make the voice")
	}
}

// At the provider's limit the voice on the key unused longest goes, and the
// new one is made in its place.
func TestVoiceClearsAwayTheOldestAtTheLimit(t *testing.T) {
	store := &adapters.MemoryVoices{}
	if v, _ := runVoice(t, "old voice", &mock.Voices{}, store); v == "" {
		t.Fatal("no first voice")
	}
	full := &limitOnce{}
	v, j := runVoice(t, "new voice", full, store)
	if v == "" || !strings.Contains(kinds(j), "cleared") || full.deleted == "" {
		t.Errorf("at the limit: %q, %s, deleted %q", v, kinds(j), full.deleted)
	}
}

// limitOnce refuses its first voice for the limit, then makes them.
type limitOnce struct {
	mock.Voices
	refused bool
	deleted string
}

func (l *limitOnce) Create(ctx context.Context, name, description string) (string, error) {
	if !l.refused {
		l.refused = true
		return "", errors.New("voices api: 400 Bad Request: voice limit reached")
	}
	return l.Voices.Create(ctx, name, description)
}

func (l *limitOnce) Delete(_ context.Context, id string) error {
	l.deleted = id
	return nil
}

// Before a voice is made, voices of ours the store no longer knows go, as
// nothing can use them; a voice made outside ChatGameLab stays.
func TestVoiceClearsOrphansAndSparesForeignVoices(t *testing.T) {
	maker := &mock.Voices{}
	maker.Hold("voice_orphan", adapters.VoiceNamePrefix+"lost")
	maker.Hold("voice_foreign", "My own narrator")

	v, j := runVoice(t, "deep", maker, &adapters.MemoryVoices{})
	if v == "" || !strings.Contains(kinds(j), "cleared-orphan") {
		t.Errorf("got %q, %s", v, kinds(j))
	}
	if maker.Has("voice_orphan") {
		t.Error("the orphan of ours is still there")
	}
	if !maker.Has("voice_foreign") {
		t.Error("a voice made outside ChatGameLab was deleted")
	}
}

// At the cap, the voice of ours unused longest goes before a new one is made.
func TestVoiceAtTheCapDeletesTheLeastRecentlyUsed(t *testing.T) {
	maker, store := &mock.Voices{}, &adapters.MemoryVoices{}
	old, _ := runVoiceCapped(t, "first", maker, store, 2)
	recent, _ := runVoiceCapped(t, "second", maker, store, 2)
	// The first voice is used again, so the second is now unused longest.
	if again, _ := runVoiceCapped(t, "first", maker, store, 2); again != old {
		t.Fatalf("the first voice was not reused: %q", again)
	}

	third, j := runVoiceCapped(t, "third", maker, store, 2)
	if third == "" || !strings.Contains(kinds(j), "cleared") {
		t.Fatalf("got %q, %s", third, kinds(j))
	}
	if maker.Has(recent) || !maker.Has(old) {
		t.Errorf("deleted the wrong voice: still has the unused %v, lost the used %v", maker.Has(recent), !maker.Has(old))
	}
}

// An account full of voices made elsewhere keeps them all; the game speaks
// in the built-in voice, and the journal says why.
func TestVoiceNeverDeletesForeignVoicesAtTheLimit(t *testing.T) {
	maker := &mock.Voices{Limit: 1}
	maker.Hold("voice_foreign", "My own narrator")

	v, j := runVoice(t, "deep", maker, &adapters.MemoryVoices{})
	if v != "stone" || !strings.Contains(kinds(j), "create-failed") {
		t.Errorf("got %q, %s", v, kinds(j))
	}
	said := ""
	for _, e := range j {
		said += e.Detail
	}
	if !strings.Contains(said, "outside ChatGameLab") {
		t.Errorf("the journal does not say why: %s", said)
	}
	if !maker.Has("voice_foreign") {
		t.Error("a voice made outside ChatGameLab was deleted")
	}
}

// A voice begun is made and kept even when the session that began it ends
// meanwhile, so the next session reuses it.
func TestVoiceIsKeptWhenItsSessionEnds(t *testing.T) {
	maker, store := &mock.Voices{Pace: 200 * time.Millisecond}, &adapters.MemoryVoices{}
	slot := &adapters.VoicesSlot{}
	slot.Set(maker, store)
	b := NewVoice("voice", "deep and slow", slot)
	b.Makes = true
	b.UseJournal(ports.NewJournal(100, nil))
	b.TextInPort() <- "a deep, slow voice"
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel() // the session ends while the voice is being made

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if id, _, _ := store.Lookup(context.Background(), StyleHash("deep and slow")); id != "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the voice begun was not kept after its session ended")
}

// The voice is made from the brief, not from the author's words, and kept
// under the author's words, so the same note reuses it.
func TestVoiceIsMadeFromTheBrief(t *testing.T) {
	maker := &recordingMaker{}
	store := &adapters.MemoryVoices{}
	v, _ := runVoice(t, "ein Fantasy Troll", maker, store)
	if v == "" || maker.description != "ein Fantasy Troll, as a voice" {
		t.Errorf("made %q from %q, want it made from the brief", v, maker.description)
	}
	if id, _, _ := store.Lookup(context.Background(), StyleHash("ein Fantasy Troll")); id != v {
		t.Errorf("kept under the author's words as %q, want %q", id, v)
	}
}

type recordingMaker struct {
	mock.Voices
	description string
}

func (r *recordingMaker) Create(ctx context.Context, name, description string) (string, error) {
	r.description = description
	return r.Voices.Create(ctx, name, description)
}
