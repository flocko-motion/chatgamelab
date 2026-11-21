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
-- workshops. The organisation shown is their role's, or their workshop's.

-- name: V2Users :many
-- The users the caller may see, for a list or for one user (id set): the
-- List grants on UserCard for a list, the Read grants for one user.
SELECT u.id, u.name, u.email, u.language, u.age_group, u.ai_quality_tier,
  u.default_api_key_share_id, u.guest_token, u.created_at, u.modified_at,
  r.role,
  o.id   AS organisation_id,
  o.name AS organisation_name,
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
  -- gate
  AND (  (@card_own::bool          AND u.id = sqlc.narg(caller)::uuid)
      OR (@card_organisation::bool AND (r.institution_id = sqlc.narg(own_organisation)::uuid OR w.institution_id = sqlc.narg(own_organisation)::uuid))
      OR (@card_member::bool       AND r.institution_id = sqlc.narg(own_organisation)::uuid
                                   AND r.role IN ('individual', 'staff', 'head'))
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
