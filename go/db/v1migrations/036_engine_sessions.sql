-- Migration 036: sessions run by the engine
-- The platform stores the engine's sessions: its bookkeeping beside the spec,
-- state and event log the engine hands it. Games name the genre they are
-- played as; existing games are adventures.

ALTER TABLE game ADD COLUMN IF NOT EXISTS genre text NOT NULL DEFAULT 'adventure';

CREATE TABLE IF NOT EXISTS game_session (
    id              uuid PRIMARY KEY,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_at     timestamptz NOT NULL DEFAULT now(),

    game_id         uuid NOT NULL REFERENCES game(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    workshop_id     uuid NULL REFERENCES workshop(id) ON DELETE SET NULL,
    game_share_id   uuid NULL REFERENCES game_share(id) ON DELETE SET NULL,
    api_key_share_id uuid NULL REFERENCES api_key_share(id) ON DELETE SET NULL,
    platform        text NOT NULL,
    language        text NOT NULL DEFAULT 'en',
    messages_sent   integer NOT NULL DEFAULT 0,

    schema_version  integer NOT NULL,
    spec            jsonb NOT NULL,
    state           jsonb NOT NULL
);
CREATE INDEX IF NOT EXISTS game_session_user_idx ON game_session (user_id, modified_at DESC);
CREATE INDEX IF NOT EXISTS game_session_game_idx ON game_session (game_id);
CREATE INDEX IF NOT EXISTS game_session_modified_idx ON game_session (modified_at);

CREATE TABLE IF NOT EXISTS game_session_event (
    session_id      uuid NOT NULL REFERENCES game_session(id) ON DELETE CASCADE,
    seq             bigint NOT NULL,
    stream          text NOT NULL,
    value           text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, seq)
);
