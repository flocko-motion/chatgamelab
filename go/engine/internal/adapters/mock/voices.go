package mock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"engine/internal/adapters"
)

// Voices makes a voice per description, named after it, so a test can tell
// one made from another, and holds what it made as the provider would; Made
// counts them, Fail makes every Create fail, Limit refuses past that many.
type Voices struct {
	Pace  Pace
	Fail  error
	Limit int
	Made  atomic.Int32

	mu   sync.Mutex
	held []adapters.VoiceInfo
}

// Hold puts a voice in the account as if made elsewhere, such as a foreign
// voice or an orphan of ours.
func (v *Voices) Hold(id, name string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.held = append(v.held, adapters.VoiceInfo{ID: id, Name: name, CreatedAt: time.Now()})
}

func (v *Voices) Create(ctx context.Context, name, description string) (string, error) {
	wait(ctx, v.Pace)
	if v.Fail != nil {
		return "", v.Fail
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.Limit > 0 && len(v.held) >= v.Limit {
		return "", errLimit
	}
	v.Made.Add(1)
	sum := sha256.Sum256([]byte(description))
	id := "voice_mock_" + hex.EncodeToString(sum[:4])
	v.held = append(v.held, adapters.VoiceInfo{ID: id, Name: name, CreatedAt: time.Now()})
	return id, nil
}

func (v *Voices) Delete(_ context.Context, id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	for i, h := range v.held {
		if h.ID == id {
			v.held = append(v.held[:i], v.held[i+1:]...)
			break
		}
	}
	return nil
}

func (v *Voices) List(context.Context) ([]adapters.VoiceInfo, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]adapters.VoiceInfo(nil), v.held...), nil
}

// Has reports whether the account still holds a voice.
func (v *Voices) Has(id string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, h := range v.held {
		if h.ID == id {
			return true
		}
	}
	return false
}

type limitError struct{}

func (limitError) Error() string { return "voices api: 400 Bad Request: voice limit reached" }

var errLimit error = limitError{}
