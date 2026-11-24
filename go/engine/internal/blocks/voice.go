package blocks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"engine/internal/adapters"
	"engine/internal/ports"
)

// Voice gives a game's voice style a voice of its own on the session's key:
// the one made for it before, or a new one, made once and kept for every
// later session on that key. It has two outputs: the voice's id, for the
// conversation to speak in, when it has one; and the voice style itself, as
// text, always, so the conversation knows how the voice is meant to sound,
// and a built-in voice follows it when no voice could be made. It reports
// working, then ready, so a gate can wait for it.
//
// While another session makes the same voice, it waits for that one rather
// than make a second. The provider keeps a limited number of voices, so
// before it makes one it clears room: first voices of ours the store no
// longer knows, which nothing can use; then, while ours on the key are at
// the cap, the one unused longest. A voice the account's owner made
// elsewhere is never deleted.
type Voice struct {
	name  string
	style string
	slot  *adapters.VoicesSlot

	// Wait is how long it waits for another session's voice, Poll how often
	// it looks; set before Start.
	Wait, Poll time.Duration
	// Cap is how many voices of ours a key keeps before the one unused
	// longest goes: under the provider's 100, so voices made elsewhere fit.
	Cap int

	out     ports.TextBroadcast
	styled  ports.TextBroadcast
	state   ports.StateBroadcast
	journal *ports.Journal
}

// NewVoice makes the block for a voice style; an empty one makes no voice.
func NewVoice(name, style string, slot *adapters.VoicesSlot) *Voice {
	return &Voice{name: name, style: strings.TrimSpace(style), slot: slot, Wait: 30 * time.Second, Poll: 500 * time.Millisecond, Cap: 90}
}

func (b *Voice) NodeName() string { return b.name }

// TextOutPort is the voice, by id; SecondaryTextOutPort the voice style.
func (b *Voice) TextOutPort() <-chan string          { return b.out.Subscribe() }
func (b *Voice) SecondaryTextOutPort() <-chan string { return b.styled.Subscribe() }
func (b *Voice) StateOutPort() <-chan ports.State    { return b.state.Subscribe() }
func (b *Voice) UseJournal(j *ports.Journal)         { b.journal = j }

// voiceMaking is how long making a voice may take, whatever happens to the
// session that began it.
const voiceMaking = 90 * time.Second

// StyleHash names a voice style for the store: the same words, the same voice.
func StyleHash(style string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(style)))
	return hex.EncodeToString(sum[:])
}

func (b *Voice) Start(ctx context.Context) {
	go func() {
		defer func() { b.out.Close(); b.styled.Close(); b.state.Close() }()
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
		// How the voice is meant to sound, whatever voice speaks it.
		if b.style != "" {
			b.styled.Send(b.style)
		}
		// A voice made before is reused without showing work: nothing is made.
		if id := b.madeBefore(ctx); id != "" {
			b.out.Send(id)
			b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady, Done: true})
			return
		}
		// Owed to a gate whatever happens, or the game never starts.
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseWorking})
		if id := b.voice(ctx); id != "" {
			b.out.Send(id)
		}
		b.state.Send(ports.State{Node: b.name, Phase: ports.PhaseReady})
	}()
}

// madeBefore is the voice made before for the style on this key, "" when
// there is none or nothing to ask; the journal says when one is reused.
func (b *Voice) madeBefore(ctx context.Context) string {
	if b.style == "" {
		return ""
	}
	maker, store := b.slot.Get()
	if maker == nil || store == nil {
		return ""
	}
	hash := StyleHash(b.style)
	id, _, err := store.Lookup(ctx, hash)
	if err != nil || id == "" {
		return ""
	}
	_ = store.Touch(ctx, hash)
	b.journal.Note(b.name, "reused", "the voice made before for this voice style on this key: "+id)
	return id
}

// voice is the id to speak in, or "" for the built-in voice; the journal
// says which and why.
func (b *Voice) voice(ctx context.Context) string {
	if b.style == "" {
		b.journal.Note(b.name, "built-in", "the game has no voice style: the built-in voice speaks")
		return ""
	}
	maker, store := b.slot.Get()
	if maker == nil || store == nil {
		b.journal.Note(b.name, "built-in", "this platform makes no voices: the built-in voice speaks, as the voice style says")
		return ""
	}
	hash := StyleHash(b.style)

	if id, _, err := store.Lookup(ctx, hash); err == nil && id != "" {
		_ = store.Touch(ctx, hash)
		b.journal.Note(b.name, "reused", "the voice made before for this voice style on this key: "+id)
		return id
	}

	claimed, err := store.Claim(ctx, hash)
	if err != nil {
		b.journal.Warn(b.name, "store-failed", "the voices could not be read: "+err.Error()+"; the built-in voice speaks")
		return ""
	}
	if !claimed {
		return b.awaitOther(ctx, store, hash)
	}

	// A voice once begun is finished and kept, even when this session ends
	// meanwhile: the next session on the key reuses it rather than start over.
	making, cancel := context.WithTimeout(context.WithoutCancel(ctx), voiceMaking)
	defer cancel()
	started := time.Now()
	id, err := b.create(making, maker, store, hash)
	if err != nil {
		_ = store.Release(making, hash)
		b.journal.Warn(b.name, "create-failed", "no voice could be made: "+err.Error()+"; the built-in voice speaks, as the voice style says")
		return ""
	}
	if err := store.Record(making, hash, id); err != nil {
		b.journal.Warn(b.name, "store-failed", "the new voice "+id+" could not be kept: "+err.Error())
	}
	b.journal.Note(b.name, "created", fmt.Sprintf("a new voice for this voice style on this key, made in %.1f s: %s", time.Since(started).Seconds(), id))
	return id
}

// create makes the voice, clearing room first; if the provider still
// refuses for its limit, it clears away one more voice of ours and tries
// once more.
func (b *Voice) create(ctx context.Context, maker adapters.Voices, store adapters.VoiceStore, hash string) (string, error) {
	b.clearRoom(ctx, maker, store)
	name := adapters.VoiceNamePrefix + hash[:12]
	id, err := maker.Create(ctx, name, b.style)
	if err == nil || !atLimit(err) {
		return id, err
	}
	if !b.evictOne(ctx, maker, store) {
		return "", fmt.Errorf("%w; the account's voices were made outside ChatGameLab, and none of them is deleted", err)
	}
	return maker.Create(ctx, name, b.style)
}

// clearRoom deletes voices of ours nothing can use, those the store no
// longer knows, then, while ours on the key are at the cap, the one unused
// longest. Without the provider's list it only counts what the store knows.
func (b *Voice) clearRoom(ctx context.Context, maker adapters.Voices, store adapters.VoiceStore) {
	known, err := store.IDs(ctx)
	if err != nil {
		return
	}
	kept := map[string]bool{}
	for _, id := range known {
		kept[id] = true
	}
	if listed, err := maker.List(ctx); err == nil {
		for _, v := range listed {
			// Ours by name, yet unknown to the store: an orphan.
			if strings.HasPrefix(v.Name, adapters.VoiceNamePrefix) && !kept[v.ID] {
				if maker.Delete(ctx, v.ID) == nil {
					b.journal.Note(b.name, "cleared-orphan", "a voice of ours no session can find any more went: "+v.ID)
				}
			}
		}
	}
	for n := len(known); n >= b.Cap; n-- {
		if !b.evictOne(ctx, maker, store) {
			return
		}
	}
}

// evictOne deletes the voice of ours on the key unused longest; false when
// there is none to delete.
func (b *Voice) evictOne(ctx context.Context, maker adapters.Voices, store adapters.VoiceStore) bool {
	old, ok, err := store.Evict(ctx)
	if err != nil || !ok {
		return false
	}
	if err := maker.Delete(ctx, old); err != nil {
		b.journal.Warn(b.name, "clear-failed", "the voice "+old+" could not be deleted: "+err.Error())
		return false
	}
	b.journal.Note(b.name, "cleared", "room for a new voice: the voice of ours unused longest went, "+old)
	return true
}

// atLimit reports whether a refusal says the account holds all the voices
// it may; the provider documents no code for it, so its words decide.
func atLimit(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "limit") || strings.Contains(msg, "maximum") || strings.Contains(msg, "too many")
}

// awaitOther waits for the voice another session is making.
func (b *Voice) awaitOther(ctx context.Context, store adapters.VoiceStore, hash string) string {
	b.journal.Note(b.name, "waiting", "another session is making this voice; waiting for it")
	deadline := time.Now().Add(b.Wait)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ""
		case <-time.After(b.Poll):
		}
		id, pending, err := store.Lookup(ctx, hash)
		switch {
		case err != nil:
			continue
		case id != "":
			_ = store.Touch(ctx, hash)
			b.journal.Note(b.name, "reused", "the voice another session made: "+id)
			return id
		case !pending:
			// The other session gave up; this one does not try again now.
			b.journal.Warn(b.name, "create-failed", "the other session made no voice; the built-in voice speaks, as the voice style says")
			return ""
		}
	}
	b.journal.Warn(b.name, "timeout", "the voice was not ready in time; the built-in voice speaks, as the voice style says")
	return ""
}
