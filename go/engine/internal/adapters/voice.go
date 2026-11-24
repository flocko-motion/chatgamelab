package adapters

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Voices makes custom voices from a description, on the provider account
// behind a key. A voice made once is kept and used by its id.
type Voices interface {
	// Create makes a voice from a description and answers its id.
	Create(ctx context.Context, name, description string) (string, error)
	// Delete removes a voice the provider holds.
	Delete(ctx context.Context, id string) error
	// List answers every voice the provider holds for the account, ours and
	// any other; an error when it cannot say.
	List(ctx context.Context) ([]VoiceInfo, error)
}

// VoiceInfo is a voice as the provider lists it: no last use, only when it
// was made.
type VoiceInfo struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// BuiltinVoice is a provider's own voice, by name, as its catalogue presents
// it: the language it speaks natively (a BCP 47 primary tag), its accent, and
// its presentation, "feminine" or "masculine", empty when not stated.
type BuiltinVoice struct {
	Name     string
	Language string
	Accent   string
	Gender   string
}

// VoiceNamePrefix names the voices the engine makes, so they are told from
// voices the account's owner made elsewhere, which the engine never deletes.
const VoiceNamePrefix = "cgl:"

// VoiceStore keeps the custom voices one game has on one key, by the hash of
// the voice style each was made from, so every session on that key reuses
// them. The platform keeps them; the engine only asks.
type VoiceStore interface {
	// Lookup answers the voice made for a style; pending says a session is
	// making it now. Neither: there is none.
	Lookup(ctx context.Context, styleHash string) (voiceID string, pending bool, err error)
	// Claim makes the caller the one making the voice for a style; false
	// when another session holds the claim or the voice exists.
	Claim(ctx context.Context, styleHash string) (bool, error)
	// Record keeps the voice a claim made.
	Record(ctx context.Context, styleHash, voiceID string) error
	// Release gives up a claim that made no voice.
	Release(ctx context.Context, styleHash string) error
	// Touch marks a voice used now.
	Touch(ctx context.Context, styleHash string) error
	// Evict removes the voice on the key unused longest, of any game, and
	// answers its id, for the provider to delete; false when there is none.
	Evict(ctx context.Context) (voiceID string, ok bool, err error)
	// IDs answers the voices kept for the key, by the provider's id.
	IDs(ctx context.Context) ([]string, error)
}

// VoiceSamples is a VoiceStore that also keeps each voice's sample, and
// removes one voice by its style, so an editor can hear a voice without the
// provider speaking it again, and replace a voice that does not fit.
type VoiceSamples interface {
	VoiceStore
	// Remove removes the voice made for a style, and its sample, and answers
	// its id, for the provider to delete; false when there is none.
	Remove(ctx context.Context, styleHash string) (voiceID string, ok bool, err error)
	// Sample answers the sample kept for the voice made for a style.
	Sample(ctx context.Context, styleHash string) (audio []byte, contentType string, ok bool, err error)
	// KeepSample keeps a sample for the voice made for a style; nothing when
	// the voice is gone, and a sample kept before stays.
	KeepSample(ctx context.Context, styleHash string, audio []byte, contentType string) error
}

// SayRequest is one line for a voice to say, outside any game: the voice (a
// made voice's id, a built-in voice's name, or empty for the default), how to
// say it, and the line.
type SayRequest struct {
	Voice        string
	Instructions string
	Line         string
}

// SayRate is the sample rate of what a Sayer answers: 16-bit signed
// little-endian mono PCM.
const SayRate = 24000

// Sayer has a voice say one line and answers the audio, as PCM at SayRate.
type Sayer interface {
	Say(ctx context.Context, req SayRequest) ([]byte, Usage, error)
}

// VoicesSlot holds the settled offer's voice maker and the store for its
// key; either is nil where the platform makes no voices or keeps none.
type VoicesSlot struct {
	held atomic.Pointer[voicesHeld]
}

type voicesHeld struct {
	maker Voices
	store VoiceStore
}

// Set replaces both; the next voice is made and kept with them.
func (s *VoicesSlot) Set(maker Voices, store VoiceStore) {
	s.held.Store(&voicesHeld{maker: maker, store: store})
}

// Get answers what was set; nil and nil before anything was.
func (s *VoicesSlot) Get() (Voices, VoiceStore) {
	held := s.held.Load()
	if held == nil {
		return nil, nil
	}
	return held.maker, held.store
}

// MemoryVoices is a VoiceStore in memory, for a dev server and tests: it
// keeps voices as long as the process runs.
type MemoryVoices struct {
	mu      sync.Mutex
	voices  map[string]string
	pending map[string]bool
	order   []string
	samples map[string]memorySample
}

type memorySample struct {
	audio       []byte
	contentType string
}

func (m *MemoryVoices) init() {
	if m.voices == nil {
		m.voices, m.pending, m.samples = map[string]string{}, map[string]bool{}, map[string]memorySample{}
	}
}

func (m *MemoryVoices) Lookup(_ context.Context, hash string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	return m.voices[hash], m.pending[hash], nil
}

func (m *MemoryVoices) Claim(_ context.Context, hash string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	if m.voices[hash] != "" || m.pending[hash] {
		return false, nil
	}
	m.pending[hash] = true
	return true, nil
}

func (m *MemoryVoices) Record(_ context.Context, hash, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	delete(m.pending, hash)
	m.voices[hash] = id
	m.touch(hash)
	return nil
}

func (m *MemoryVoices) Release(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	delete(m.pending, hash)
	return nil
}

func (m *MemoryVoices) Touch(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touch(hash)
	return nil
}

func (m *MemoryVoices) Evict(context.Context) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	for len(m.order) > 0 {
		hash := m.order[0]
		m.order = m.order[1:]
		if id := m.voices[hash]; id != "" {
			delete(m.voices, hash)
			delete(m.samples, hash)
			return id, true, nil
		}
	}
	return "", false, nil
}

func (m *MemoryVoices) IDs(context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []string
	for _, id := range m.voices {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (m *MemoryVoices) Remove(_ context.Context, hash string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	id := m.voices[hash]
	if id == "" {
		return "", false, nil
	}
	delete(m.voices, hash)
	delete(m.samples, hash)
	return id, true, nil
}

func (m *MemoryVoices) Sample(_ context.Context, hash string) ([]byte, string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	s, ok := m.samples[hash]
	return s.audio, s.contentType, ok, nil
}

func (m *MemoryVoices) KeepSample(_ context.Context, hash string, audio []byte, contentType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	if _, kept := m.samples[hash]; kept || m.voices[hash] == "" {
		return nil
	}
	m.samples[hash] = memorySample{audio: audio, contentType: contentType}
	return nil
}

// touch moves a voice to the back of the order: the front is unused longest.
func (m *MemoryVoices) touch(hash string) {
	for i, h := range m.order {
		if h == hash {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	m.order = append(m.order, hash)
}
