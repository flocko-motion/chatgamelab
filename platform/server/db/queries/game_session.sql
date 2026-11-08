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

-- name: GetGameSessionByID :one
SELECT * FROM game_session WHERE id = $1;

-- name: SaveGameSessionState :execrows
UPDATE game_session SET
  schema_version = $2,
  spec = $3,
  state = $4,
  modified_at = now()
WHERE id = $1;

-- name: AppendGameSessionEvent :exec
INSERT INTO game_session_event (session_id, seq, stream, value)
VALUES (
  $1,
  (SELECT COALESCE(MAX(seq), 0) + 1 FROM game_session_event WHERE session_id = $1),
  $2, $3
);

-- name: TouchGameSession :exec
UPDATE game_session SET modified_at = now() WHERE id = $1;

-- name: GetGameSessionEvents :many
SELECT stream, value FROM game_session_event WHERE session_id = $1 ORDER BY seq ASC;

-- name: CountGameSessionMessage :exec
UPDATE game_session SET messages_sent = messages_sent + 1, modified_at = now() WHERE id = $1;

-- name: GetUserGameSessions :many
-- A player's sessions, newest activity first, each with its game's name.
SELECT gs.id, gs.game_id, gs.user_id, gs.workshop_id, gs.platform, gs.language,
       gs.messages_sent, gs.created_at, gs.modified_at, g.name AS game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1
  AND g.deleted_at IS NULL
  AND (sqlc.arg(search)::text = '' OR LOWER(g.name) LIKE LOWER('%' || sqlc.arg(search)::text || '%'))
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::text = 'game' THEN LOWER(g.name) END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'model' THEN gs.platform END ASC,
  gs.modified_at DESC
LIMIT 20;

-- name: GetGameSessionsByGameID :many
SELECT id, game_id, user_id, workshop_id, platform, language, messages_sent, created_at, modified_at
FROM game_session WHERE game_id = $1 ORDER BY modified_at DESC;

-- name: GetUserGameSessionIDsForGame :many
SELECT id FROM game_session WHERE user_id = $1 AND game_id = $2;

-- name: DeleteGameSession :exec
DELETE FROM game_session WHERE id = $1;

-- name: DeleteGameSessionsByGameID :many
DELETE FROM game_session WHERE game_id = $1 RETURNING id;

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
