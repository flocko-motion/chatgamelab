-- app_user -------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO app_user (id, name, email, auth0_id, age_group, language)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
RETURNING id;

-- name: CreateUserWithID :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: GetUserIDByAuth0ID :one
SELECT id FROM app_user WHERE auth0_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM app_user WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByGuestToken :one
-- Get user by participant token, but only if they're linked to an active workshop
SELECT u.*
FROM app_user u
INNER JOIN user_role ur ON u.id = ur.user_id
INNER JOIN workshop w ON ur.workshop_id = w.id
WHERE u.guest_token = $1
  AND u.deleted_at IS NULL
  AND ur.role = 'guest'
  AND w.active = true
  AND w.deleted_at IS NULL;

-- name: CheckGuestTokenStatus :one
-- Check if a participant token exists and get the workshop active status
-- Returns: exists (bool), workshop_active (bool)
SELECT
  EXISTS(
    SELECT 1 FROM app_user u
    INNER JOIN user_role ur ON u.id = ur.user_id
    WHERE u.guest_token = $1
      AND u.deleted_at IS NULL
      AND ur.role = 'guest'
  ) AS token_exists,
  COALESCE(
    (SELECT w.active FROM app_user u
     INNER JOIN user_role ur ON u.id = ur.user_id
     INNER JOIN workshop w ON ur.workshop_id = w.id
     WHERE u.guest_token = $1
       AND u.deleted_at IS NULL
       AND ur.role = 'guest'
       AND w.deleted_at IS NULL
     LIMIT 1),
    false
  ) AS workshop_active;

-- name: GuestTokenExists :one
-- Unfiltered: soft-deleted users keep their token, so it must never be reissued.
SELECT EXISTS(SELECT 1 FROM app_user WHERE guest_token = $1) AS taken;

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
  u.age_group,
  r.id           AS role_id,
  r.role         AS role,
  r.institution_id,
  i.name         AS institution_name,
  i.free_use_api_key_share_id AS institution_free_use_api_key_share_id,
  i.prompt_constraints AS institution_prompt_constraints,
  r.workshop_id,
  w.name         AS workshop_name,
  w.show_public_games AS workshop_show_public_games,
  w.show_other_participants_games AS workshop_show_other_participants_games,
  w.ai_quality_tier AS workshop_ai_quality_tier,
  w.prompt_constraints AS workshop_prompt_constraints,
  w.is_paused AS workshop_is_paused,
  w.allow_game_sharing AS workshop_allow_game_sharing,
  r.active_workshop_id,
  aw.name        AS active_workshop_name,
  aw.show_public_games AS active_workshop_show_public_games,
  aw.show_other_participants_games AS active_workshop_show_other_participants_games,
  aw.ai_quality_tier AS active_workshop_ai_quality_tier,
  aw.prompt_constraints AS active_workshop_prompt_constraints,
  aw.is_paused AS active_workshop_is_paused,
  aw.allow_game_sharing AS active_workshop_allow_game_sharing
FROM app_user u
LEFT JOIN user_role r ON r.id = (
  SELECT ur.id
  FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY CASE WHEN ur.role = 'guest' THEN 1 ELSE 0 END, ur.created_at DESC
  LIMIT 1
)
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
LEFT JOIN user_role r ON r.id = (
  SELECT ur.id
  FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY CASE WHEN ur.role = 'guest' THEN 1 ELSE 0 END, ur.created_at DESC
  LIMIT 1
)
LEFT JOIN institution i
  ON i.id = r.institution_id
LEFT JOIN workshop w
  ON w.id = r.workshop_id
WHERE u.deleted_at IS NULL
ORDER BY u.created_at DESC;

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

-- name: UpdateUserAgeGroup :exec
UPDATE app_user SET
  age_group = $2,
  modified_at = now()
WHERE id = $1;

-- name: UpdateUserLanguage :exec
UPDATE app_user SET
  language = $2,
  modified_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM app_user WHERE id = $1;

-- user_role -------------------------------------------------------------

-- name: DeleteUserRoles :exec
DELETE FROM user_role WHERE user_id = $1;

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

-- name: SetDefaultApiKey :exec
UPDATE api_key SET is_default = true, modified_at = now()
WHERE id = $1 AND user_id = $2;

-- name: GetApiKeyByID :one
SELECT * FROM api_key WHERE id = $1;

-- name: GetApiKeyIDsByUser :many
SELECT id FROM api_key WHERE user_id = $1;

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

-- name: CountUserGames :one
SELECT COUNT(*)::int AS count FROM game WHERE created_by = $1;

-- name: SumPlayCountOfUserGames :one
SELECT COALESCE(SUM(play_count), 0)::int AS total FROM game WHERE created_by = $1;

-- User deletion cleanup queries

-- name: DeleteUserFavorites :exec
DELETE FROM user_favourite_game WHERE user_id = $1;

-- name: CountHeadsByInstitution :one
-- Count how many heads an institution has (for last-head protection)
SELECT COUNT(*)::int AS count
FROM user_role
WHERE institution_id = $1 AND role = 'head';

-- name: CountAdmins :one
-- Count live admins (for the ADMIN_EMAILS bootstrap gate)
SELECT COUNT(*)::int AS count
FROM user_role r
JOIN app_user u ON u.id = r.user_id
WHERE r.role = 'admin' AND u.deleted_at IS NULL;

-- name: DeleteInvitesForUser :exec
-- Delete invites targeting this user (can't just null invited_user_id due to type check constraint)
DELETE FROM user_role_invite WHERE invited_user_id = $1;

-- name: ClearInviteAcceptedByUser :exec
UPDATE user_role_invite SET accepted_by = NULL
WHERE accepted_by = $1;

-- name: ClearGameOriginalCreator :exec
-- Clear originally_created_by references to a deleted user
UPDATE game SET originally_created_by = NULL
WHERE originally_created_by = $1;

-- name: CreateAccountForAuth0 :one
-- An account from its first sign-in: no name yet, so registration is incomplete.
INSERT INTO app_user (id, auth0_id)
VALUES (gen_random_uuid(), $1)
ON CONFLICT (auth0_id) DO NOTHING
RETURNING id;

-- name: GetGuestStatus :one
-- has_token marks a participant account; it may sign in while it holds the
-- participant role in an active workshop.
SELECT
  (u.guest_token IS NOT NULL)::bool AS has_token,
  EXISTS(
    SELECT 1 FROM user_role ur
    WHERE ur.user_id = u.id AND ur.role = 'guest'
  ) AS has_role,
  EXISTS(
    SELECT 1 FROM user_role ur
    JOIN workshop w ON w.id = ur.workshop_id
    WHERE ur.user_id = u.id AND ur.role = 'guest'
      AND w.active AND w.deleted_at IS NULL
  ) AS workshop_active
FROM app_user u
WHERE u.id = $1;
