-- v2: the secure core's queries for sessions (#344). core/session.go has
-- checked that the caller may; these read and write what they are told.

-- name: V2SessionFacts :one
-- What the session guard decides on: whose a session is (a guest who came
-- through a share link plays it by its id), its game, and the state of the
-- workshop and the organisation it belongs to: its workshop's, or its share
-- link's.
SELECT s.id, s.user_id, s.game_id, s.workshop_id, s.game_share_id,
  (u.private_share_id IS NOT NULL)::bool                AS link_guest,
  (g.id IS NULL OR g.deleted_at IS NOT NULL)::bool       AS game_gone,
  COALESCE(w.is_paused, false)::bool                     AS workshop_paused,
  (o.id IS NOT NULL AND (o.suspended_at IS NOT NULL
           OR institution_consent_missing(o.id, @declaration_version::text, @declaration_hash::text)))::bool AS organisation_suspended
FROM game_session s
JOIN app_user u        ON u.id = s.user_id
LEFT JOIN game g       ON g.id = s.game_id
LEFT JOIN workshop w   ON w.id = s.workshop_id
LEFT JOIN game_share gs ON gs.id = s.game_share_id
LEFT JOIN institution o ON o.id = COALESCE(w.institution_id, gs.institution_id)
WHERE s.id = @id;

-- name: V2WorkshopPaused :one
-- Whether a workshop is paused; a workshop gone is not.
SELECT COALESCE((SELECT is_paused FROM workshop WHERE id = @id AND deleted_at IS NULL), false)::bool;

-- name: V2SessionOwner :one
SELECT user_id FROM game_session WHERE id = @id;

-- name: V2DeleteSession :exec
DELETE FROM game_session WHERE id = @id;

-- name: V2OwnGameSessions :many
-- An account's sessions of one game, newest activity first.
SELECT * FROM game_session WHERE user_id = @user_id AND game_id = @game_id ORDER BY modified_at DESC;

-- name: V2RuleFacts :one
-- What the youth-protection rule reads about a player (or a share link's
-- author): their age group, whether they came through a share link, and the
-- rules and names of the workshop they are in and of their organisation.
SELECT u.age_group, (u.private_share_id IS NOT NULL)::bool AS link_guest,
  w.name AS workshop_name, w.prompt_constraints AS workshop_rule, wo.name AS workshop_organisation_name,
  o.name AS organisation_name, o.prompt_constraints AS organisation_rule
FROM app_user u
LEFT JOIN user_role r ON r.id = (
  SELECT ur.id FROM user_role ur WHERE ur.user_id = u.id
  ORDER BY CASE WHEN ur.role = 'guest' THEN 1 ELSE 0 END, ur.created_at DESC
  LIMIT 1
)
LEFT JOIN workshop w    ON w.id = CASE WHEN r.role = 'guest' THEN r.workshop_id ELSE r.active_workshop_id END
LEFT JOIN institution wo ON wo.id = w.institution_id
-- their organisation (head, staff), or the one of the workshop they take part in
LEFT JOIN institution o ON o.id = COALESCE(r.institution_id, w.institution_id)
WHERE u.id = @id;

-- name: V2ShareLinkPlay :many
-- A share link as a play through it needs it: its game, workshop and
-- organisation, plays left, tier, author and paying share, and whether its
-- organisation (its own, or its workshop's) is suspended.
SELECT gs.id, gs.game_id, gs.workshop_id, gs.institution_id, gs.remaining, gs.ai_quality_tier,
  gs.created_by, gs.api_key_share_id, w.ai_quality_tier AS workshop_tier,
  (o.id IS NOT NULL AND (o.suspended_at IS NOT NULL
     OR institution_consent_missing(o.id, @declaration_version::text, @declaration_hash::text)))::bool AS organisation_suspended
FROM game_share gs
LEFT JOIN workshop w    ON w.id = gs.workshop_id
LEFT JOIN institution o ON o.id = COALESCE(gs.institution_id, w.institution_id)
WHERE (sqlc.narg(token)::text IS NULL OR gs.token = sqlc.narg(token)::text)
  AND (sqlc.narg(id)::uuid IS NULL OR gs.id = sqlc.narg(id)::uuid);

-- name: V2RuleSource :one
-- A workshop's or an organisation's rule and name, for a share link's.
SELECT w.name AS workshop_name, w.prompt_constraints AS workshop_rule, wo.name AS workshop_organisation_name
FROM workshop w LEFT JOIN institution wo ON wo.id = w.institution_id
WHERE w.id = @id;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2CountGameSessionMessage :exec
UPDATE game_session SET messages_sent = messages_sent + 1, modified_at = now() WHERE id = $1;

-- name: V2CountUserGameSessions :one
SELECT COUNT(*)::int AS count FROM game_session WHERE user_id = $1;

-- name: V2DeleteGameSessionsByGameID :many
DELETE FROM game_session WHERE game_id = $1 RETURNING id;

-- name: V2GetGameSessionByID :one
SELECT * FROM game_session WHERE id = $1;

-- name: V2GetUserGameSessions :many
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

-- name: V2SumUserGameSessionMessages :one
SELECT COALESCE(SUM(messages_sent), 0)::int AS total FROM game_session WHERE user_id = $1;

-- The engine's session store (core/internal/sessionstore), carried over from v1.

-- name: V2CreateGameSession :exec
-- The first state the engine saves creates the row.
INSERT INTO game_session (
  id, game_id, user_id, workshop_id, game_share_id, api_key_share_id,
  platform, language, schema_version, spec, state
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11
);

-- name: V2CreateGuestUser :one
INSERT INTO app_user (id, name, email, auth0_id, guest_token, private_share_id)
VALUES ($1, $2, NULL, NULL, NULL, $3)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- User Statistics queries

-- name: V2DecrementGameShareRemaining :one
-- Atomically decrements the remaining counter. Returns the share if successful.
-- Succeeds when: remaining is NULL (unlimited) or remaining > 0.
UPDATE game_share SET remaining = CASE
  WHEN remaining IS NULL THEN NULL
  ELSE remaining - 1
END
WHERE id = $1 AND (remaining IS NULL OR remaining > 0) RETURNING *;

-- name: V2DeleteGameSession :exec
DELETE FROM game_session WHERE id = $1;

-- name: V2GetUserGameSessionIDsForGame :many
SELECT id FROM game_session WHERE user_id = $1 AND game_id = $2;

-- name: V2IncrementGamePlayCount :exec
UPDATE game SET play_count = play_count + 1 WHERE id = $1;

-- name: V2AppendGameSessionEvent :exec
INSERT INTO game_session_event (session_id, seq, stream, value)
VALUES (
  $1,
  (SELECT COALESCE(MAX(seq), 0) + 1 FROM game_session_event WHERE session_id = $1),
  $2, $3
);

-- name: V2GetGameSessionEvents :many
SELECT stream, value FROM game_session_event WHERE session_id = $1 ORDER BY seq ASC;

-- name: V2SaveGameSessionState :execrows
UPDATE game_session SET
  schema_version = $2,
  spec = $3,
  state = $4,
  modified_at = now()
WHERE id = $1;

-- name: V2TouchGameSession :exec
UPDATE game_session SET modified_at = now() WHERE id = $1;

-- Carried over from v1 for the core's system jobs (#344).

-- name: V2PurgeExpiredSessions :many
-- At most a batch of the sessions idle since the cutoff, so one statement
-- holds its locks briefly.
DELETE FROM game_session gs WHERE gs.id IN (
  SELECT old.id FROM game_session old WHERE old.modified_at < $1 ORDER BY old.modified_at LIMIT $2
) RETURNING gs.id;
