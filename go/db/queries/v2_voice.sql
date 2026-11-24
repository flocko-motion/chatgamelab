-- v2: the voices made on a key (core/voice.go). A voice is keyed by the key
-- and the hash of the voice style it was made from, whichever game asks.

-- name: V2Voice :one
-- The voice made on a key for a voice style; voice_id is NULL while a session
-- is making it.
SELECT voice_id, claimed_at, created_at
FROM voice
WHERE api_key_id = @api_key_id::uuid AND style_hash = @style_hash::text;

-- name: V2ClaimVoice :one
-- Claims making the voice for a voice style on a key: a new claim, or one
-- left longer than stale_seconds without a voice. Answers the row when the
-- claim is the caller's; no row when another session holds it or the voice
-- exists.
INSERT INTO voice (api_key_id, style_hash, claimed_at)
VALUES (@api_key_id::uuid, @style_hash::text, now())
ON CONFLICT (api_key_id, style_hash) DO UPDATE SET claimed_at = now()
  WHERE voice.voice_id IS NULL
    AND voice.claimed_at < now() - make_interval(secs => @stale_seconds::int)
RETURNING claimed_at;

-- name: V2SetVoice :exec
-- Records the voice a claim made; it is used from now on.
UPDATE voice SET voice_id = @voice_id::text, created_at = now(), last_used_at = now()
WHERE api_key_id = @api_key_id::uuid AND style_hash = @style_hash::text;

-- name: V2ReleaseVoice :exec
-- Gives up a claim that made no voice, so the next session may try.
DELETE FROM voice
WHERE api_key_id = @api_key_id::uuid AND style_hash = @style_hash::text AND voice_id IS NULL;

-- name: V2TouchVoice :exec
-- Marks a voice used now, so clearing away keeps it longest.
UPDATE voice SET last_used_at = now()
WHERE api_key_id = @api_key_id::uuid AND style_hash = @style_hash::text;

-- name: V2EvictVoice :one
-- Removes the voice on a key unused longest and answers its id, for the
-- provider to delete too; no row when the key has none.
DELETE FROM voice
WHERE ctid = (
  SELECT ctid FROM voice
  WHERE api_key_id = @api_key_id::uuid AND voice_id IS NOT NULL
  ORDER BY last_used_at NULLS FIRST
  LIMIT 1
)
RETURNING voice_id;

-- name: V2VoiceIDs :many
-- The voices made on a key, by the provider's id: what is ours among the
-- voices the provider holds for the account.
SELECT voice_id::text FROM voice
WHERE api_key_id = @api_key_id::uuid AND voice_id IS NOT NULL;

-- name: V2RemoveVoice :one
-- Removes the voice made on a key for a voice style, its sample with it, and
-- answers its id, for the provider to delete too; no row when there is none.
-- A claim (no voice yet) stays: a session is making that voice now.
DELETE FROM voice
WHERE api_key_id = @api_key_id::uuid AND style_hash = @style_hash::text AND voice_id IS NOT NULL
RETURNING voice_id;

-- name: V2VoiceSample :one
-- The sample of the voice made on a key for a voice style.
SELECT audio, content_type
FROM voice_sample
WHERE api_key_id = @api_key_id::uuid AND style_hash = @style_hash::text;

-- name: V2SetVoiceSample :exec
-- Keeps a voice's sample; the voice must exist, and a sample made before
-- stays (it keeps the language it was first made in).
INSERT INTO voice_sample (api_key_id, style_hash, audio, content_type)
SELECT v.api_key_id, v.style_hash, @audio::bytea, @content_type::text
FROM voice v
WHERE v.api_key_id = @api_key_id::uuid AND v.style_hash = @style_hash::text AND v.voice_id IS NOT NULL
ON CONFLICT (api_key_id, style_hash) DO NOTHING;
