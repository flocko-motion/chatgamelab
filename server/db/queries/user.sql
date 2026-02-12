-- app_user -------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id;

-- name: CreateUserWithID :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: CreateUserWithParticipantToken :one
INSERT INTO app_user (id, name, email, auth0_id, participant_token)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: GetUserIDByAuth0ID :one
SELECT id FROM app_user WHERE auth0_id = $1;

-- name: GetUserByAuth0ID :one
SELECT * FROM app_user WHERE auth0_id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM app_user WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByParticipantToken :one
-- Get user by participant token, but only if they're linked to an active workshop
SELECT u.*
FROM app_user u
INNER JOIN user_role ur ON u.id = ur.user_id
INNER JOIN workshop w ON ur.workshop_id = w.id
WHERE u.participant_token = $1
  AND u.deleted_at IS NULL
  AND ur.role = 'participant'
  AND w.active = true
  AND w.deleted_at IS NULL;

-- name: CheckParticipantTokenStatus :one
-- Check if a participant token exists and get the workshop active status
-- Returns: exists (bool), workshop_active (bool)
SELECT
  EXISTS(
    SELECT 1 FROM app_user u
    INNER JOIN user_role ur ON u.id = ur.user_id
    WHERE u.participant_token = $1
      AND u.deleted_at IS NULL
      AND ur.role = 'participant'
  ) AS token_exists,
  COALESCE(
    (SELECT w.active FROM app_user u
     INNER JOIN user_role ur ON u.id = ur.user_id
     INNER JOIN workshop w ON ur.workshop_id = w.id
     WHERE u.participant_token = $1
       AND u.deleted_at IS NULL
       AND ur.role = 'participant'
       AND w.deleted_at IS NULL
     LIMIT 1),
    false
  ) AS workshop_active;

-- name: IsNameTaken :one
SELECT EXISTS(SELECT 1 FROM app_user WHERE name = $1 AND deleted_at IS NULL) AS taken;

-- name: IsNameTakenByOther :one
SELECT EXISTS(SELECT 1 FROM app_user WHERE name = $1 AND id != $2 AND deleted_at IS NULL) AS taken;

-- name: IsEmailTakenByOther :one
SELECT EXISTS(SELECT 1 FROM app_user WHERE email = $1 AND id != $2 AND deleted_at IS NULL) AS taken;

-- name: GetUserByID :one
SELECT * FROM app_user WHERE id = $1;

-- name: GetUserDetailsByID :one
SELECT
  u.id,
  u.created_by,
  u.created_at,
  u.modified_by,
  u.modified_at,
  u.name,
  u.email,
  u.deleted_at,
  u.auth0_id,
  u.default_api_key_share_id,
  u.ai_quality_tier,
  u.language,
  r.id           AS role_id,
  r.role         AS role,
  r.institution_id,
  i.name         AS institution_name,
  i.free_use_api_key_share_id AS institution_free_use_api_key_share_id,
  r.workshop_id,
  w.name         AS workshop_name,
  w.show_public_games AS workshop_show_public_games,
  w.show_other_participants_games AS workshop_show_other_participants_games,
  w.ai_quality_tier AS workshop_ai_quality_tier,
  w.design_editing_enabled AS workshop_design_editing_enabled,
  w.is_paused AS workshop_is_paused,
  r.active_workshop_id,
  aw.name        AS active_workshop_name,
  aw.show_public_games AS active_workshop_show_public_games,
  aw.show_other_participants_games AS active_workshop_show_other_participants_games,
  aw.ai_quality_tier AS active_workshop_ai_quality_tier,
  aw.design_editing_enabled AS active_workshop_design_editing_enabled,
  aw.is_paused AS active_workshop_is_paused
FROM app_user u
LEFT JOIN LATERAL (
  SELECT ur.*
  FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY ur.created_at DESC
  LIMIT 1
) r ON TRUE
LEFT JOIN institution i
  ON i.id = r.institution_id
LEFT JOIN workshop w
  ON w.id = r.workshop_id
LEFT JOIN workshop aw
  ON aw.id = r.active_workshop_id
WHERE u.id = $1;

-- name: GetAllUsersWithDetails :many
SELECT
  u.id,
  u.created_by,
  u.created_at,
  u.modified_by,
  u.modified_at,
  u.name,
  u.email,
  u.deleted_at,
  u.auth0_id,
  r.id           AS role_id,
  r.role         AS role,
  r.institution_id,
  i.name         AS institution_name,
  r.workshop_id,
  w.name         AS workshop_name
FROM app_user u
LEFT JOIN LATERAL (
  SELECT ur.*
  FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY ur.created_at DESC
  LIMIT 1
) r ON TRUE
LEFT JOIN institution i
  ON i.id = r.institution_id
LEFT JOIN workshop w
  ON w.id = r.workshop_id
WHERE u.deleted_at IS NULL
ORDER BY u.created_at DESC;

-- name: GetUserApiKeys :many
SELECT
  k.id,
  k.created_by,
  k.created_at,
  k.modified_by,
  k.modified_at,
  k.user_id,
  u.name AS user_name,
  k.name,
  k.platform,
  k.key
FROM api_key k
JOIN app_user u ON u.id = k.user_id
WHERE k.user_id = $1;

-- name: UpdateUser :exec
UPDATE app_user SET
  name = $2,
  email = $3,
  modified_at = now()
WHERE id = $1;

-- name: UpdateUserAiQualityTier :exec
UPDATE app_user SET
  ai_quality_tier = $2,
  modified_at = now()
WHERE id = $1;

-- name: UpdateUserLanguage :exec
UPDATE app_user SET
  language = $2,
  modified_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
UPDATE app_user
SET
  deleted_at = now()
WHERE id = $1;

-- user_role -------------------------------------------------------------

-- name: DeleteUserRoles :exec
DELETE FROM user_role WHERE user_id = $1;

-- name: DeleteUserRole :exec
DELETE FROM user_role WHERE user_id = $1;

-- name: GetUsersByInstitution :many
SELECT
  u.id, u.name, u.email,
  u.created_by, u.created_at, u.modified_by, u.modified_at,
  ur.id as role_id, ur.role as role_role
FROM app_user u
INNER JOIN user_role ur ON u.id = ur.user_id
WHERE ur.institution_id = $1
  AND u.deleted_at IS NULL
  AND ur.role IN ('individual', 'staff', 'head');

-- name: CreateUserRole :one
INSERT INTO user_role (id, user_id, role, institution_id, workshop_id)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING id;

-- name: IsUserInWorkshop :one
-- Check if a user is a member of a specific workshop (has a user_role with that workshop_id)
SELECT EXISTS(
  SELECT 1 FROM user_role ur
  INNER JOIN workshop w ON w.id = ur.workshop_id
  WHERE ur.user_id = $1
    AND ur.workshop_id = $2
    AND w.active = true
    AND w.deleted_at IS NULL
) AS is_member;

-- name: CanUserAccessShareViaWorkshopDefault :one
-- Check if a user can access an API key share because it's the default share for a workshop they're in
-- Matches both participants (workshop_id) and individuals in workshop mode (active_workshop_id)
SELECT EXISTS(
  SELECT 1 FROM workshop w
  INNER JOIN user_role ur ON (ur.workshop_id = w.id OR ur.active_workshop_id = w.id)
  WHERE w.default_api_key_share_id = $1
    AND ur.user_id = $2
    AND w.active = true
    AND w.deleted_at IS NULL
) AS can_access;

-- active_workshop ------------------------------------------------------

-- name: SetUserActiveWorkshop :exec
UPDATE user_role SET active_workshop_id = $2, modified_at = now()
WHERE user_id = $1;

-- name: ClearUserActiveWorkshop :exec
UPDATE user_role SET active_workshop_id = NULL, modified_at = now()
WHERE user_id = $1;

-- name: GetUserActiveWorkshopID :one
-- Get the active workshop ID for a user (for game creation context)
SELECT active_workshop_id FROM user_role
WHERE user_id = $1 AND active_workshop_id IS NOT NULL
LIMIT 1;

-- api_key --------------------------------------------------------------

-- name: CreateApiKey :one
INSERT INTO api_key (
  id, created_by,
  created_at, modified_by, modified_at,
  user_id, name, platform, key, is_default
) VALUES (
  gen_random_uuid(), $1,
  $2, $3, $4,
  $5, $6, $7, $8, $9
)
RETURNING *;

-- name: CountApiKeysByUser :one
SELECT COUNT(*) FROM api_key WHERE user_id = $1;

-- name: ClearDefaultApiKey :exec
UPDATE api_key SET is_default = false, modified_at = now()
WHERE user_id = $1 AND is_default = true;

-- name: SetDefaultApiKey :exec
UPDATE api_key SET is_default = true, modified_at = now()
WHERE id = $1 AND user_id = $2;

-- name: GetDefaultApiKey :one
SELECT * FROM api_key WHERE user_id = $1 AND is_default = true;

-- name: UpdateApiKeyLastUsageSuccess :exec
UPDATE api_key SET last_usage_success = $2, modified_at = now()
WHERE id = $1;

-- name: GetApiKeyByID :one
SELECT * FROM api_key WHERE id = $1;

-- name: UpdateApiKey :one
UPDATE api_key SET
  modified_by = $2,
  modified_at = $3,
  name = $4
WHERE id = $1
RETURNING *;

-- name: GetApiKeyIDsByUser :many
SELECT id FROM api_key WHERE user_id = $1;

-- name: DeleteApiKey :exec
DELETE FROM api_key WHERE id = $1 AND user_id = $2;

-- name: DeleteAllApiKeysByUser :exec
DELETE FROM api_key WHERE user_id = $1;

-- GetApiKeySharesByUserID is now in api_key.sql using the unified api_key_share table

-- name: SetUserDefaultApiKeyShare :exec
UPDATE app_user SET
  default_api_key_share_id = $2,
  modified_at = now()
WHERE id = $1;

-- name: GetUserDefaultApiKeyShare :one
SELECT default_api_key_share_id FROM app_user WHERE id = $1;

-- user_role_invite -------------------------------------------------------------

-- name: CreateTargetedInvite :one
INSERT INTO user_role_invite (
  id, created_by, created_at, modified_by, modified_at,
  institution_id, role, workshop_id,
  invited_user_id, invited_email,
  invite_token,
  status
) VALUES (
  gen_random_uuid(), $1, now(), $1, now(),
  $2, $3, $4,
  $5, $6,
  $7,
  'pending'
)
RETURNING *;

-- name: CreateOpenInvite :one
INSERT INTO user_role_invite (
  id, created_by, created_at, modified_by, modified_at,
  institution_id, role, workshop_id,
  invite_token, max_uses, expires_at,
  status
) VALUES (
  gen_random_uuid(), $1, now(), $1, now(),
  $2, $3, $4,
  $5, $6, $7,
  'pending'
)
RETURNING *;

-- name: GetInvites :many
SELECT * FROM user_role_invite
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetInvitesByUser :many
SELECT * FROM user_role_invite
WHERE (invited_user_id = $1 OR invited_email = (SELECT email FROM app_user WHERE id = $1))
  AND deleted_at IS NULL
  AND status = 'pending'
ORDER BY created_at DESC;

-- name: GetInvitesByWorkshop :many
SELECT * FROM user_role_invite
WHERE workshop_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetInvitesByInstitution :many
SELECT * FROM user_role_invite
WHERE institution_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: HasWorkshopRole :one
SELECT EXISTS(
    SELECT 1 FROM user_role
    WHERE user_id = $1 AND workshop_id = $2
) AS has_role;

-- name: GetUsersByWorkshop :many
SELECT
  u.id, u.name, u.email,
  u.created_by, u.created_at, u.modified_by, u.modified_at,
  u.deleted_at, u.auth0_id,
  ur.id as role_id, ur.role, ur.institution_id, ur.workshop_id
FROM app_user u
INNER JOIN user_role ur ON u.id = ur.user_id
WHERE ur.workshop_id = $1
  AND u.deleted_at IS NULL;

-- name: GetWorkshopParticipants :many
-- Get all participants for a workshop, including:
-- 1. Users with RoleParticipant (anonymous participants)
-- 2. Workshop owner/creator (staff/head who created it)
SELECT
  u.id, u.name, u.auth0_id,
  COALESCE(ur.created_at, w.created_at) as joined_at,
  COALESCE(ur.role, ur_inst.role) as role,
  (SELECT COUNT(*) FROM game g WHERE g.created_by = u.id AND g.deleted_at IS NULL)::int as games_count
FROM app_user u
INNER JOIN workshop w ON w.id = $1
LEFT JOIN user_role ur ON u.id = ur.user_id AND ur.workshop_id = $1 AND ur.role = 'participant'
LEFT JOIN user_role ur_inst ON u.id = ur_inst.user_id AND ur_inst.workshop_id IS NULL AND u.id = w.created_by
WHERE (ur.user_id IS NOT NULL OR u.id = w.created_by)
  AND u.deleted_at IS NULL
ORDER BY joined_at ASC;

-- name: GetInviteByID :one
SELECT * FROM user_role_invite WHERE id = $1;

-- name: GetInviteByToken :one
SELECT * FROM user_role_invite WHERE invite_token = $1;

-- name: UpdateInviteStatus :exec
UPDATE user_role_invite SET
  status = $2,
  modified_at = now()
WHERE id = $1;

-- name: AcceptTargetedInvite :exec
UPDATE user_role_invite SET
  status = 'accepted',
  accepted_at = now(),
  accepted_by = $2,
  modified_at = now()
WHERE id = $1;

-- name: IncrementInviteUses :exec
UPDATE user_role_invite SET
  uses_count = uses_count + 1,
  modified_at = now()
WHERE id = $1;

-- name: DeleteInvite :exec
DELETE FROM user_role_invite WHERE id = $1;

-- name: GetPendingInviteByTarget :one
-- Check if a pending invite already exists for the same target (user_id or email) and institution
SELECT * FROM user_role_invite
WHERE institution_id = $1
  AND status = 'pending'
  AND deleted_at IS NULL
  AND (
    (invited_user_id IS NOT NULL AND invited_user_id = $2)
    OR (invited_email IS NOT NULL AND invited_email = $3)
  )
LIMIT 1;

-- Guest User queries

-- name: CreateGuestUser :one
INSERT INTO app_user (id, name, email, auth0_id, participant_token, private_share_game_id)
VALUES ($1, $2, NULL, NULL, NULL, $3)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: DeleteGuestSessionMessagesByGameID :exec
-- Delete all messages belonging to sessions of guest users created via a game's share link
DELETE FROM game_session_message WHERE game_session_id IN (
  SELECT gs.id FROM game_session gs
  JOIN app_user u ON u.id = gs.user_id
  WHERE u.private_share_game_id = $1
);

-- name: DeleteGuestSessionsByGameID :exec
-- Delete all sessions of guest users created via a game's share link
DELETE FROM game_session WHERE user_id IN (
  SELECT id FROM app_user WHERE private_share_game_id = $1
);

-- name: DeleteGuestUsersByGameID :exec
-- Delete all guest users created via a game's share link
DELETE FROM app_user WHERE private_share_game_id = $1;

-- name: CountGuestUsersByGameID :one
SELECT COUNT(*)::int AS count FROM app_user WHERE private_share_game_id = $1;

-- User Statistics queries

-- name: CountUserSessions :one
SELECT COUNT(*)::int AS count FROM game_session WHERE user_id = $1;

-- name: CountUserGames :one
SELECT COUNT(*)::int AS count FROM game WHERE created_by = $1;

-- name: CountUserPlayerMessages :one
SELECT COUNT(*)::int AS count
FROM game_session_message m
JOIN game_session s ON s.id = m.game_session_id
WHERE s.user_id = $1 AND m.type = 'player';

-- name: SumPlayCountOfUserGames :one
SELECT COALESCE(SUM(play_count), 0)::int AS total FROM game WHERE created_by = $1;

-- User deletion cleanup queries

-- name: DeleteUserFavourites :exec
DELETE FROM user_favourite_game WHERE user_id = $1;

-- name: DeleteWorkshopParticipantsByUserID :exec
DELETE FROM workshop_participant WHERE workshop_id IN (
  SELECT workshop_id FROM user_role WHERE user_id = $1 AND role = 'participant' AND workshop_id IS NOT NULL
);

-- name: CountHeadsByInstitution :one
-- Count how many heads an institution has (for last-head protection)
SELECT COUNT(*)::int AS count
FROM user_role
WHERE institution_id = $1 AND role = 'head';

-- name: GetParticipantUserIDsByWorkshopID :many
-- Get user IDs of participants in a workshop
SELECT user_id FROM user_role WHERE workshop_id = $1 AND role = 'participant';

-- name: DeleteUserRolesByWorkshopID :exec
-- Delete all participant roles scoped to a workshop
DELETE FROM user_role WHERE workshop_id = $1 AND role = 'participant';
