-- v2: the secure core's user queries (#344).
--
-- Every relation fragment appears once per section; the grant table only
-- switches it on, through one boolean per (section, relation). Which grant
-- opens which fragment is in core/gates.go.
--
-- A user's role is chosen as everywhere (V2ReadCaller): one role per
-- account, a guest's last, newest first. "Their workshop" is the one they
-- are in: a guest's own, or the one entered in workshop mode. A user is "of
-- own organisation" when their role belongs to the caller's organisation or
-- they are in one of its workshops (v1); a guest of it is in one of its
-- workshops. The organisation shown is their role's, or their workshop's;
-- the one they are a member of is their role's only (head and staff): taking
-- part in a workshop is no membership.

-- name: V2Users :many
-- The users the caller may see, for a list or for one user (id set): the
-- List grants on UserCard for a list, the Read grants for one user.
SELECT u.id, u.name, u.email, u.language, u.age_group, u.ai_quality_tier,
  u.default_api_key_share_id, u.guest_token, u.created_at, u.modified_at,
  (u.registered_at IS NOT NULL)::bool AS registered,
  r.role,
  o.id   AS organisation_id,
  o.name AS organisation_name,
  r.institution_id AS member_of,
  w.id   AS workshop_id,
  w.name AS workshop_name,
  -- the sections: NULL (no caller, no organisation) is false
  COALESCE(
     (@account_own::bool          AND u.id = sqlc.narg(caller)::uuid)
  OR (@account_organisation::bool AND (r.institution_id = sqlc.narg(own_organisation)::uuid OR w.institution_id = sqlc.narg(own_organisation)::uuid))
  OR  @account_everywhere::bool, false)::bool                         AS may_read_account,
  COALESCE(
     (@contact_own::bool          AND u.id = sqlc.narg(caller)::uuid)
  OR (@contact_organisation::bool AND (r.institution_id = sqlc.narg(own_organisation)::uuid OR w.institution_id = sqlc.narg(own_organisation)::uuid))
  OR  @contact_everywhere::bool, false)::bool                         AS may_read_contact,
  COALESCE(
     (@secrets_own::bool          AND u.id = sqlc.narg(caller)::uuid)
  OR (@secrets_organisation::bool AND (r.institution_id = sqlc.narg(own_organisation)::uuid OR w.institution_id = sqlc.narg(own_organisation)::uuid)),
     false)::bool                                                     AS may_read_secrets,
  -- the actions
  COALESCE(
     (@edit_card_own::bool        AND u.id = sqlc.narg(caller)::uuid)
  OR (@edit_card_guest::bool      AND r.role = 'guest' AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_card_everywhere::bool, false)::bool                       AS may_edit_card,
  COALESCE(
     (@edit_account_own::bool     AND u.id = sqlc.narg(caller)::uuid)
  OR (@edit_account_guest::bool   AND r.role = 'guest' AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_account_everywhere::bool, false)::bool                    AS may_edit_account,
  COALESCE(
     (@edit_contact_own::bool     AND u.id = sqlc.narg(caller)::uuid)
  OR  @edit_contact_everywhere::bool, false)::bool                    AS may_edit_contact,
  COALESCE(
     -- a guest with an Auth0 sign-in became one through an email invite (a
     -- v1 bug, .tasks/email-invite-keeps-role.md): never deleted as a guest
     (@delete_guest::bool         AND r.role = 'guest' AND w.institution_id = sqlc.narg(own_organisation)::uuid
                                  AND u.auth0_id IS NULL)
  OR  @delete_everywhere::bool, false)::bool                          AS may_delete
FROM app_user u
LEFT JOIN user_role r ON r.id = (
  SELECT ur.id FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY CASE WHEN ur.role = 'guest' THEN 1 ELSE 0 END, ur.created_at DESC
  LIMIT 1
)
LEFT JOIN workshop w    ON w.id = CASE WHEN r.role = 'guest' THEN r.workshop_id ELSE r.active_workshop_id END
LEFT JOIN institution o ON o.id = COALESCE(r.institution_id, w.institution_id)
WHERE u.deleted_at IS NULL
  AND (sqlc.narg(id)::uuid IS NULL OR u.id = sqlc.narg(id)::uuid)
  -- a wish: the members of one organisation (head and staff)
  AND (sqlc.narg(members_of)::uuid IS NULL
       OR (r.institution_id = sqlc.narg(members_of)::uuid AND r.role IN ('staff', 'head')))
  -- gate
  AND (  (@card_own::bool          AND u.id = sqlc.narg(caller)::uuid)
      OR (@card_organisation::bool AND (r.institution_id = sqlc.narg(own_organisation)::uuid OR w.institution_id = sqlc.narg(own_organisation)::uuid))
      OR (@card_member::bool       AND r.institution_id = sqlc.narg(own_organisation)::uuid
                                   AND r.role IN ('staff', 'head'))
      OR  @card_everywhere::bool )
ORDER BY u.created_at DESC, u.id;

-- name: V2UpdateUser :exec
-- Writes a user's card, account and contact as the caller changed them;
-- core/user_update.go has checked that they may.
UPDATE app_user SET
  modified_by              = @modified_by::uuid,
  modified_at              = now(),
  -- card
  name                     = @name::text,
  -- account
  language                 = @language::text,
  age_group                = sqlc.narg(age_group)::text,
  ai_quality_tier          = sqlc.narg(ai_quality_tier)::text,
  default_api_key_share_id = sqlc.narg(default_api_key_share_id)::uuid,
  -- contact
  email                    = sqlc.narg(email)::text
WHERE id = @id::uuid AND deleted_at IS NULL;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2ClearUserActiveWorkshop :exec
UPDATE user_role SET active_workshop_id = NULL, modified_at = now()
WHERE user_id = $1 AND role != 'guest';

-- name: V2CompleteRegistration :execrows
-- Registration completes the account, once: registered_at ends its sign-up.
UPDATE app_user SET
  name = $2,
  email = $3,
  age_group = $4,
  language = COALESCE(NULLIF(sqlc.arg(language)::text, ''), language),
  registered_at = now(),
  modified_at = now()
WHERE id = $1 AND name IS NULL AND registered_at IS NULL;

-- name: V2CountHeadsByInstitution :one
-- Count how many heads an institution has (for last-head protection)
SELECT COUNT(*)::int AS count
FROM user_role
WHERE institution_id = $1 AND role = 'head';

-- name: V2CountUserGames :one
SELECT COUNT(*)::int AS count FROM game WHERE created_by = $1;

-- name: V2CreateUserRole :one
INSERT INTO user_role (id, user_id, role, institution_id, workshop_id)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING id;

-- name: V2CreateUserWithGuestToken :one
INSERT INTO app_user (id, name, email, auth0_id, guest_token, language)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: V2DeleteUserGuestRole :exec
-- Delete a specific user's participant role for a workshop
DELETE FROM user_role WHERE user_id = $1 AND workshop_id = $2 AND role = 'guest';

-- name: V2DeleteUserRoles :exec
DELETE FROM user_role WHERE user_id = $1;

-- name: V2GetUserByEmail :one
SELECT * FROM app_user WHERE email = $1 AND deleted_at IS NULL;

-- name: V2GetUserByID :one
SELECT * FROM app_user WHERE id = $1;

-- name: V2GetWorkshopParticipants :many
-- Get all members currently associated with a workshop:
-- 1. Participants (permanent members with workshop_id)
-- 2. Workshop creator (always shown)
-- 3. Visitors with active_workshop_id set (individuals, head, staff)
-- Uses DISTINCT ON to deduplicate users who match multiple conditions,
-- preferring their non-participant role (individual/head/staff) over participant.
SELECT * FROM (
  SELECT DISTINCT ON (u.id)
    u.id, u.name, u.auth0_id,
    COALESCE(ur.created_at, w.created_at) as joined_at,
    ur.role,
    (SELECT COUNT(*) FROM game g WHERE g.created_by = u.id AND g.deleted_at IS NULL)::int as games_count,
    CASE
      WHEN u.id = w.created_by THEN true
      WHEN ur.workshop_id = $1 AND ur.role = 'guest' THEN true
      ELSE false
    END as permanent
  FROM app_user u
  INNER JOIN workshop w ON w.id = $1
  INNER JOIN user_role ur ON u.id = ur.user_id
  WHERE u.deleted_at IS NULL
    AND (
      (ur.workshop_id = $1)
      OR (ur.active_workshop_id = $1 AND ur.role != 'guest')
      OR (u.id = w.created_by AND ur.role != 'guest')
    )
  ORDER BY u.id,
    CASE WHEN ur.role = 'guest' THEN 1 ELSE 0 END,
    ur.created_at ASC
) members
ORDER BY joined_at ASC;

-- name: V2IsEmailTakenByOther :one
SELECT EXISTS(SELECT 1 FROM app_user WHERE email = $1 AND id != $2 AND deleted_at IS NULL) AS taken;

-- name: V2IsNameTaken :one
SELECT EXISTS(SELECT 1 FROM app_user WHERE name = $1 AND deleted_at IS NULL) AS taken;

-- name: V2IsNameTakenByOther :one
SELECT EXISTS(SELECT 1 FROM app_user WHERE name = $1 AND id != $2 AND deleted_at IS NULL) AS taken;

-- name: V2SetUserActiveWorkshop :exec
UPDATE user_role SET active_workshop_id = $2, modified_at = now()
WHERE user_id = $1 AND role != 'guest';

-- name: V2SumPlayCountOfUserGames :one
SELECT COALESCE(SUM(play_count), 0)::int AS total FROM game WHERE created_by = $1;

-- User deletion cleanup queries

-- name: V2UpdateGuestToken :execrows
-- Only replaces an existing token, never mints one: a registered participant has a
-- real login, and a word code would be a second, weaker credential for it. The row
-- count tells the caller whether there was anything to replace.
UPDATE app_user SET guest_token = $2, modified_by = $3, modified_at = now()
WHERE id = $1 AND guest_token IS NOT NULL;

-- name: V2GuestTokenExists :one
-- Unfiltered: soft-deleted users keep their token, so it is never reissued.
SELECT EXISTS(SELECT 1 FROM app_user WHERE guest_token = @token)::bool AS taken;

-- Deleting an account (core/delete.go): what hangs on it, then the account.

-- name: V2GamesCreatedBy :many
SELECT id FROM game WHERE created_by = @user_id;

-- name: V2KeysOf :many
SELECT id FROM api_key WHERE user_id = @user_id;

-- name: V2ClearDefaultShare :exec
UPDATE app_user SET default_api_key_share_id = NULL, modified_at = now() WHERE id = @user_id;

-- name: V2DeleteFavouritesOf :exec
DELETE FROM user_favourite_game WHERE user_id = @user_id;

-- name: V2DeleteInvitesFor :exec
-- Invites addressed to the user; one cannot just lose its target.
DELETE FROM user_role_invite WHERE invited_user_id = @user_id;

-- name: V2ClearInvitesAcceptedBy :exec
UPDATE user_role_invite SET accepted_by = NULL WHERE accepted_by = @user_id;

-- name: V2ClearOriginalCreator :exec
-- Copies of the user's games, owned by others, keep no pointer to them.
UPDATE game SET originally_created_by = NULL WHERE originally_created_by = @user_id;

-- name: V2DeleteAccount :exec
-- The account itself; its sessions go with it (ON DELETE CASCADE).
DELETE FROM app_user WHERE id = @user_id;

-- Carried over from v1 for the core's system jobs (#344).

-- name: V2CountAdmins :one
-- Count live admins (for the ADMIN_EMAILS bootstrap gate)
SELECT COUNT(*)::int AS count
FROM user_role r
JOIN app_user u ON u.id = r.user_id
WHERE r.role = 'admin' AND u.deleted_at IS NULL;

-- name: V2SeedUser :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: V2SeedRenameUser :exec
UPDATE app_user SET
  name = $2,
  email = $3,
  modified_at = now()
WHERE id = $1;

-- name: V2CreateAccount :one
INSERT INTO app_user (id, name, email, auth0_id, age_group, language)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
RETURNING id;

-- name: V2AccountRoles :many
-- The roles an account holds; an admin is made only of an individual.
SELECT role FROM user_role WHERE user_id = @user_id;

-- Signing in (core/signin.go).

-- name: V2Auth0Account :many
SELECT id FROM app_user WHERE auth0_id = @subject;

-- name: V2CreateAuth0Account :many
-- An account from its first sign-in, in sign-up until registration
-- completes it. A concurrent first sign-in made it already: no row.
INSERT INTO app_user (id, auth0_id, registered_at) VALUES (gen_random_uuid(), @subject, NULL)
ON CONFLICT (auth0_id) DO NOTHING
RETURNING id;

-- name: V2GuestByToken :many
-- The workshop guest a login code signs in, and whether their workshop is
-- running.
SELECT u.id, (w.active AND w.deleted_at IS NULL)::bool AS workshop_active
FROM app_user u
JOIN user_role r ON r.user_id = u.id AND r.role = 'guest'
JOIN workshop w ON w.id = r.workshop_id
WHERE u.guest_token = @token AND u.deleted_at IS NULL;

-- name: V2AccountEmail :one
SELECT email FROM app_user WHERE id = @id;

-- name: V2StaleSignUps :many
-- Accounts in sign-up whose first sign-in is older than the cutoff, with
-- what speaks against removing them: a name, a role, a session (sessions
-- go with an account; every other reference refuses its deletion).
SELECT u.id,
  (COALESCE(u.name, '') <> '')::bool AS has_name,
  EXISTS (SELECT 1 FROM user_role r WHERE r.user_id = u.id)::bool AS has_role,
  EXISTS (SELECT 1 FROM game_session s WHERE s.user_id = u.id)::bool AS has_session
FROM app_user u
WHERE u.registered_at IS NULL AND u.created_at < @cutoff
ORDER BY u.created_at
LIMIT @max_rows;

-- name: V2DeleteSignUp :execrows
-- Removes an account still in sign-up, for good: a sign-up holds no data,
-- and its Auth0 subject signs in to a fresh account.
DELETE FROM app_user u
WHERE u.id = $1 AND u.registered_at IS NULL
  AND COALESCE(u.name, '') = ''
  AND NOT EXISTS (SELECT 1 FROM user_role r WHERE r.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM game_session s WHERE s.user_id = u.id);
