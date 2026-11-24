-- A voice's sample: a few seconds of it speaking a neutral line, made on the
-- first preview and played on every later one without asking the provider
-- again. It belongs to the voice (key + voice style), not to a game, and goes
-- with it: when a new voice replaces it, and when the voice unused longest is
-- cleared away. It keeps the language it was first made in. A table of its
-- own, as game_cover, so looking a voice up at session start never loads
-- audio.
CREATE TABLE voice_sample (
    api_key_id   uuid NOT NULL,
    style_hash   text NOT NULL,
    audio        bytea NOT NULL,
    -- The audio's media type, as served (audio/wav, audio/mpeg, ...).
    content_type text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (api_key_id, style_hash),
    FOREIGN KEY (api_key_id, style_hash) REFERENCES voice (api_key_id, style_hash) ON DELETE CASCADE
);
