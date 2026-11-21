-- app_user -------------------------------------------------------------

-- name: GetUserIDByAuth0ID :one
SELECT id FROM app_user WHERE auth0_id = $1;

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

-- name: CreateAccountForAuth0 :one
-- An account from its first sign-in: no name yet, so registration is incomplete.
INSERT INTO app_user (id, auth0_id)
VALUES (gen_random_uuid(), $1)
ON CONFLICT (auth0_id) DO NOTHING
RETURNING id;

