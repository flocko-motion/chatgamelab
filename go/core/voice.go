// package: core / voices
// type:    logic
// job:     keeps the custom voices made on a key, for the engine to reuse them
// limits:  only for a payer the core chose; the engine makes and deletes the voices themselves, and speaks their samples
package core

import (
	"context"
	"database/sql"
	"errors"

	"cgl/core/internal/store"
	"cgl/db"
	"engine"

	"github.com/google/uuid"
)

// voiceClaimStale is how long a claim may stand without a voice before
// another session takes it over: the one that made it is gone.
const voiceClaimStale = 60

// VoicesFor is the store of the voices made on a payer's key, for the engine
// to reuse a voice made once for the same voice style by any game; nil for a
// payer the core did not choose, which may not spend its key.
func VoicesFor(p Payer) engine.VoiceStore {
	if !p.chosen {
		return nil
	}
	return keyVoices{key: p.KeyID}
}

type keyVoices struct{ key uuid.UUID }

func (v keyVoices) Lookup(ctx context.Context, hash string) (string, bool, error) {
	row, err := store.Queries().V2Voice(ctx, db.V2VoiceParams{ApiKeyID: v.key, StyleHash: hash})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.VoiceID.String, !row.VoiceID.Valid, nil
}

func (v keyVoices) Claim(ctx context.Context, hash string) (bool, error) {
	_, err := store.Queries().V2ClaimVoice(ctx, db.V2ClaimVoiceParams{ApiKeyID: v.key, StyleHash: hash, StaleSeconds: voiceClaimStale})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (v keyVoices) Record(ctx context.Context, hash, voiceID string) error {
	return store.Queries().V2SetVoice(ctx, db.V2SetVoiceParams{VoiceID: voiceID, ApiKeyID: v.key, StyleHash: hash})
}

func (v keyVoices) Release(ctx context.Context, hash string) error {
	return store.Queries().V2ReleaseVoice(ctx, db.V2ReleaseVoiceParams{ApiKeyID: v.key, StyleHash: hash})
}

func (v keyVoices) Touch(ctx context.Context, hash string) error {
	return store.Queries().V2TouchVoice(ctx, db.V2TouchVoiceParams{ApiKeyID: v.key, StyleHash: hash})
}

func (v keyVoices) Evict(ctx context.Context) (string, bool, error) {
	id, err := store.Queries().V2EvictVoice(ctx, v.key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id.String, id.Valid, nil
}

func (v keyVoices) IDs(ctx context.Context) ([]string, error) {
	return store.Queries().V2VoiceIDs(ctx, v.key)
}

// Remove removes the voice made on the key for a style, and its sample with
// it, and answers its id, for the provider to delete; false when there is
// none. A claim stays: a session is making that voice now.
func (v keyVoices) Remove(ctx context.Context, hash string) (string, bool, error) {
	id, err := store.Queries().V2RemoveVoice(ctx, db.V2RemoveVoiceParams{ApiKeyID: v.key, StyleHash: hash})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id.String, id.Valid, nil
}

// Sample answers the sample kept for the voice made on the key for a style,
// and its media type; false when there is none.
func (v keyVoices) Sample(ctx context.Context, hash string) ([]byte, string, bool, error) {
	row, err := store.Queries().V2VoiceSample(ctx, db.V2VoiceSampleParams{ApiKeyID: v.key, StyleHash: hash})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	return row.Audio, row.ContentType, true, nil
}

// KeepSample keeps a sample for the voice made on the key for a style. It
// keeps nothing when the voice is gone, and a sample kept before stays.
func (v keyVoices) KeepSample(ctx context.Context, hash string, audio []byte, contentType string) error {
	return store.Queries().V2SetVoiceSample(ctx, db.V2SetVoiceSampleParams{
		ApiKeyID: v.key, StyleHash: hash, Audio: audio, ContentType: contentType,
	})
}
