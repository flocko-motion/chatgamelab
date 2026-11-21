-- game_session ---------------------------------------------------------
-- Sessions the engine runs: the platform's bookkeeping, and what the engine
-- stores through its persistence interface.

-- name: CreateGameSession :exec
-- The first state the engine saves creates the row.
INSERT INTO game_session (
  id, game_id, user_id, workshop_id, game_share_id, api_key_share_id,
  platform, language, schema_version, spec, state
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11
);

-- name: PurgeExpiredGameSessions :many
-- At most a batch of the sessions idle since the cutoff, so one statement
-- holds its locks briefly.
DELETE FROM game_session gs WHERE gs.id IN (
  SELECT old.id FROM game_session old WHERE old.modified_at < $1 ORDER BY old.modified_at LIMIT $2
) RETURNING gs.id;

-- name: CountUserGameSessions :one
SELECT COUNT(*)::int AS count FROM game_session WHERE user_id = $1;

-- name: SumUserGameSessionMessages :one
SELECT COALESCE(SUM(messages_sent), 0)::int AS total FROM game_session WHERE user_id = $1;
