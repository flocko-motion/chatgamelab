-- Anonymous play statistics for the admin dashboard.
-- Game sessions and their messages are deleted 48 h after the last activity
-- (SESSION_RETENTION_HOURS); this table keeps one row per started session so
-- plays and AI turns can be counted per week and month. No account, no texts:
-- only time, game, organisation, workshop and the number of AI responses.
CREATE TABLE play_stat (
    session_id      uuid PRIMARY KEY,
    created_at      timestamptz NOT NULL DEFAULT now(),
    game_id         uuid NULL,
    institution_id  uuid NULL,
    workshop_id     uuid NULL,
    via_share       boolean NOT NULL DEFAULT false,
    turns           integer NOT NULL DEFAULT 0,
    last_turn_at    timestamptz NULL
);

CREATE INDEX play_stat_created_idx ON play_stat (created_at);
CREATE INDEX play_stat_institution_idx ON play_stat (institution_id, created_at);
