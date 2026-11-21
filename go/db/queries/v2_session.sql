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
LEFT JOIN institution o ON o.id = r.institution_id
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
