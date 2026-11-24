-- The voices made on a key: each a custom voice its provider made from a
-- voice style, kept so every later session on that key whose game has the
-- same voice style reuses it, a copied game included. A voice belongs to the
-- provider account behind the key, so it is per key, not per share; another
-- voice style is another row (style_hash).
--
-- A row without voice_id is a claim: a session is making the voice, and the
-- others wait for it rather than make a second one. A claim left too long is
-- taken over (core/voice.go).
CREATE TABLE voice (
    api_key_id   uuid NOT NULL REFERENCES api_key(id) ON DELETE CASCADE,
    style_hash   text NOT NULL,
    -- The provider's id for the voice; NULL while it is being made.
    voice_id     text NULL,
    claimed_at   timestamptz NOT NULL DEFAULT now(),
    created_at   timestamptz NULL,
    last_used_at timestamptz NULL,
    PRIMARY KEY (api_key_id, style_hash)
);

-- Clearing a key's voices away starts with the one unused longest.
CREATE INDEX voice_key_last_used ON voice (api_key_id, last_used_at);
