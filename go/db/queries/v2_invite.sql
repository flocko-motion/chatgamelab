-- v2: the secure core's queries for invites (#344). core/invite*.go has
-- checked that the caller may; these read and write what they are told.
-- An invite to become a head is a HeadInvite to the rules; every other role
-- an Invite.

-- name: V2Invites :many
-- The invites the caller may see, for a list or for one invite (id or token
-- set): the List grants on Invite and HeadInvite for a list, the Read grants
-- for one. Wishes narrow it: one organisation or workshop, those addressed to the
-- caller, pending ones only.
SELECT i.id, i.created_at, i.modified_at,
  i.institution_id, o.name AS organisation_name,
  i.role, i.workshop_id, w.name AS workshop_name,
  i.invited_user_id, i.invited_email, i.invite_token,
  i.max_uses, i.uses_count, i.expires_at, i.status, i.accepted_at, i.accepted_by,
  COALESCE(i.invited_user_id = sqlc.narg(caller)::uuid
           OR lower(i.invited_email) = lower(sqlc.narg(caller_email)::text), false)::bool AS addressed_to_caller,
  -- the actions
  COALESCE(
     (@delete_organisation::bool      AND i.institution_id = sqlc.narg(own_organisation)::uuid AND i.role <> 'head')
  OR (@delete_head_organisation::bool AND i.institution_id = sqlc.narg(own_organisation)::uuid AND i.role = 'head')
  OR (@delete_everywhere::bool        AND i.role <> 'head')
  OR (@delete_head_everywhere::bool   AND i.role = 'head'), false)::bool AS may_delete,
  COALESCE(
     (@use_mine::bool      AND i.role <> 'head' AND (i.invited_user_id = sqlc.narg(caller)::uuid
                                                     OR lower(i.invited_email) = lower(sqlc.narg(caller_email)::text)))
  OR (@use_head_mine::bool AND i.role = 'head'  AND (i.invited_user_id = sqlc.narg(caller)::uuid
                                                     OR lower(i.invited_email) = lower(sqlc.narg(caller_email)::text)))
  OR (@use_link::bool      AND i.invite_token = sqlc.narg(token)::text), false)::bool AS may_use
FROM user_role_invite i
JOIN institution o    ON o.id = i.institution_id
LEFT JOIN workshop w  ON w.id = i.workshop_id
WHERE i.deleted_at IS NULL
  AND (sqlc.narg(id)::uuid IS NULL OR i.id = sqlc.narg(id)::uuid)
  AND (sqlc.narg(token)::text IS NULL OR i.invite_token = sqlc.narg(token)::text)
  -- wishes
  AND (sqlc.narg(organisation)::uuid IS NULL OR i.institution_id = sqlc.narg(organisation)::uuid)
  AND (sqlc.narg(workshop)::uuid IS NULL OR i.workshop_id = sqlc.narg(workshop)::uuid)
  AND (NOT @addressed_only::bool OR i.invited_user_id = sqlc.narg(caller)::uuid
                                 OR lower(i.invited_email) = lower(sqlc.narg(caller_email)::text))
  AND (NOT @pending_only::bool OR i.status = 'pending')
  -- gate
  AND (  (@card_organisation::bool AND i.institution_id = sqlc.narg(own_organisation)::uuid AND i.role <> 'head')
      OR (@head_organisation::bool AND i.institution_id = sqlc.narg(own_organisation)::uuid AND i.role = 'head')
      OR (@card_mine::bool AND i.role <> 'head' AND (i.invited_user_id = sqlc.narg(caller)::uuid
                                                     OR lower(i.invited_email) = lower(sqlc.narg(caller_email)::text)))
      OR (@head_mine::bool AND i.role = 'head'  AND (i.invited_user_id = sqlc.narg(caller)::uuid
                                                     OR lower(i.invited_email) = lower(sqlc.narg(caller_email)::text)))
      -- a link opens the one open invite behind it
      OR (@card_link::bool AND i.invite_token = sqlc.narg(token)::text
                           AND i.invited_user_id IS NULL AND i.invited_email IS NULL)
      OR (@card_everywhere::bool AND i.role <> 'head')
      OR (@head_everywhere::bool AND i.role = 'head') )
ORDER BY i.created_at DESC, i.id;

-- name: V2UserEmail :one
-- An account's email, to tell the invites addressed to it.
SELECT email FROM app_user WHERE id = @id AND deleted_at IS NULL;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2AcceptTargetedInvite :exec
UPDATE user_role_invite SET
  status = 'accepted',
  accepted_at = now(),
  accepted_by = $2,
  modified_at = now()
WHERE id = $1;

-- name: V2CreateOpenInvite :one
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

-- name: V2CreateTargetedInvite :one
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

-- name: V2DeleteInvite :exec
DELETE FROM user_role_invite WHERE id = $1;

-- name: V2GetInvitesByWorkshop :many
SELECT * FROM user_role_invite
WHERE workshop_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: V2GetPendingInviteByTarget :one
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

-- name: V2IncrementInviteUses :exec
UPDATE user_role_invite SET
  uses_count = uses_count + 1,
  modified_at = now()
WHERE id = $1;

-- name: V2InviteTokenExists :one
SELECT EXISTS(SELECT 1 FROM user_role_invite WHERE invite_token = $1) AS taken;

-- name: V2UpdateInviteStatus :exec
UPDATE user_role_invite SET
  status = $2,
  modified_at = now()
WHERE id = $1;
