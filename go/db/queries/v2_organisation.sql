-- v2: the secure core's queries for organisations registering themselves,
-- consenting and being suspended (#344). core/organisation_*.go has checked
-- that the caller may; these read and write what they are told.

-- name: V2OrganisationNameTaken :one
-- Any organisation's name, deleted ones included (the column is unique over
-- all rows), ignoring case.
SELECT EXISTS (SELECT 1 FROM institution WHERE lower(name) = lower(@name::text))::bool AS taken;

-- name: V2EmailOfOtherAccount :one
-- Whether an account other than @id has this email, ignoring case.
SELECT EXISTS (
  SELECT 1 FROM app_user
  WHERE lower(email) = lower(@email::text) AND id <> @id::uuid AND deleted_at IS NULL
)::bool AS taken;

-- name: V2CreateOrganisation :exec
-- An organisation with its details and restriction; an admin's may leave
-- them out (NULL).
INSERT INTO institution (
  id, created_by, modified_by, name,
  operator, street, postal_code, city, prompt_constraints
) VALUES (
  @id, @creator::uuid, @creator::uuid, @name,
  sqlc.narg(operator)::text, sqlc.narg(street)::text, sqlc.narg(postal_code)::text,
  sqlc.narg(city)::text, sqlc.narg(prompt_constraints)::text
);

-- name: V2Organisations :many
-- The organisations the caller may see, for a list or for one organisation
-- (id set): the List grants on OrganisationCard for a list, the Read grants
-- for one. Its consent log, which the record also shows, is read by
-- V2LatestConsent and V2FirstConsent.
SELECT o.id, o.name, o.created_by, o.created_at, o.modified_at,
  -- settings
  o.prompt_constraints, o.free_use_api_key_share_id, o.free_use_ai_quality_tier,
  -- details
  o.operator, o.street, o.postal_code, o.city,
  -- record
  o.suspended_at, o.suspended_reason,
  (o.suspended_at IS NOT NULL
   OR institution_consent_missing(o.id, @declaration_version::text, @declaration_hash::text))::bool AS suspended,
  -- the sections: NULL (no caller, no organisation) is false
  COALESCE(
     (@settings_organisation::bool AND o.id = sqlc.narg(own_organisation)::uuid)
  OR  @settings_everywhere::bool, false)::bool                       AS may_read_settings,
  COALESCE(
     (@details_organisation::bool  AND o.id = sqlc.narg(own_organisation)::uuid)
  OR  @details_everywhere::bool, false)::bool                        AS may_read_details,
  COALESCE(
     (@record_organisation::bool   AND o.id = sqlc.narg(own_organisation)::uuid)
  OR  @record_everywhere::bool, false)::bool                         AS may_read_record,
  -- the actions
  COALESCE(
     (@edit_card_organisation::bool     AND o.id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_card_everywhere::bool, false)::bool                      AS may_edit_card,
  COALESCE(
     (@edit_settings_organisation::bool AND o.id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_settings_everywhere::bool, false)::bool                  AS may_edit_settings,
  COALESCE(
     (@edit_details_organisation::bool  AND o.id = sqlc.narg(own_organisation)::uuid), false)::bool AS may_edit_details,
  @delete_everywhere::bool                                           AS may_delete,
  @suspend_everywhere::bool                                          AS may_suspend
FROM institution o
WHERE o.deleted_at IS NULL
  AND (sqlc.narg(id)::uuid IS NULL OR o.id = sqlc.narg(id)::uuid)
  -- gate
  AND (  (@card_organisation::bool AND o.id = sqlc.narg(own_organisation)::uuid)
      -- the organisation of the workshop the caller takes part in
      OR (@card_current_workshop::bool AND o.id = (SELECT w.institution_id FROM workshop w WHERE w.id = sqlc.narg(current_workshop)::uuid))
      OR  @card_everywhere::bool )
ORDER BY o.name, o.id;

-- name: V2UpdateOrganisation :exec
-- Writes an organisation's card and settings as the caller changed them;
-- core/organisation_update.go has checked that they may.
UPDATE institution SET
  name               = @name::text,
  prompt_constraints = sqlc.narg(prompt_constraints)::text,
  modified_by        = @modified_by::uuid,
  modified_at        = now()
WHERE id = @id::uuid AND deleted_at IS NULL;

-- name: V2UpdateOrganisationDetails :exec
-- Operator, address and youth-protection restriction, as a head confirms
-- them with the declaration.
UPDATE institution SET
  operator           = @operator::text,
  street             = @street::text,
  postal_code        = @postal_code::text,
  city               = @city::text,
  prompt_constraints = @prompt_constraints::text,
  modified_by        = @modified_by::uuid,
  modified_at        = now()
WHERE id = @id;

-- name: V2SetOrganisationSuspension :execrows
-- An admin suspends an organisation (suspended_at set) or lifts it (NULL);
-- its data stays.
UPDATE institution SET
  suspended_at     = sqlc.narg(suspended_at),
  suspended_reason = sqlc.narg(suspended_reason),
  modified_by      = @modified_by::uuid,
  modified_at      = now()
WHERE id = @id AND deleted_at IS NULL;

-- name: V2MakeHead :exec
-- The account's only role from now on: head of the organisation.
WITH dropped AS (DELETE FROM user_role WHERE user_id = @user_id)
INSERT INTO user_role (id, user_id, role, institution_id)
VALUES (gen_random_uuid(), @user_id, 'head', @institution_id);

-- name: V2RecordConsent :one
INSERT INTO institution_consent (
  id, institution_id, user_id, kind,
  email, person_name, person_function,
  institution_name, institution_operator, street, postal_code, city,
  declaration_version, declaration_hash
) VALUES (
  gen_random_uuid(), @institution_id, @user_id, 'consent',
  @email::text, @person_name::text, @person_function::text,
  @institution_name, @institution_operator::text, @street::text, @postal_code::text, @city::text,
  @declaration_version, @declaration_hash
)
RETURNING created_at;

-- name: V2RecordWithdrawal :one
-- Withdraws the consent before it, repeating its declaration.
INSERT INTO institution_consent (
  id, institution_id, user_id, kind, institution_name,
  declaration_version, declaration_hash
) VALUES (
  gen_random_uuid(), @institution_id, @user_id, 'withdrawal', @institution_name,
  @declaration_version, @declaration_hash
)
RETURNING created_at;

-- name: V2LatestConsent :many
-- The organisation's newest entry in the consent log, if any.
SELECT kind, declaration_version, declaration_hash, created_at
FROM institution_consent
WHERE institution_id = @institution_id
ORDER BY created_at DESC
LIMIT 1;

-- name: V2FirstConsent :many
-- The organisation's first consent, if any: given by its creator, it is how
-- the organisation registered itself.
SELECT user_id, created_at, person_name, person_function, email
FROM institution_consent
WHERE institution_id = @institution_id AND kind = 'consent'
ORDER BY created_at
LIMIT 1;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2GetInstitutionByID :one
SELECT * FROM institution WHERE id = $1 AND deleted_at IS NULL;

-- Deleting an organisation (core/delete.go): for good, with its workshops
-- and their guests; its head and staff become individuals.

-- name: V2OrganisationWorkshops :many
-- All of them, deleted ones too: they go for good.
SELECT id FROM workshop WHERE institution_id = @organisation_id;

-- name: V2OrganisationMembers :many
SELECT DISTINCT user_id FROM user_role WHERE institution_id = @organisation_id AND role IN ('head', 'staff');

-- name: V2OrganisationSharesAndLinks :many
-- What ends with the organisation, as taking shares back ends it: the
-- shares given to it, the shares drawn from them, and every share link of
-- it or its workshops, with the share each draws.
WITH org_share AS (
  SELECT k.id FROM api_key_share k WHERE k.institution_id = @organisation_id AND k.game_id IS NULL
), share AS (
  SELECT id FROM org_share
  UNION SELECT s.id FROM api_key_share s WHERE s.source_share_id IN (SELECT id FROM org_share)
  UNION SELECT gs.api_key_share_id FROM game_share gs
        LEFT JOIN workshop w ON w.id = gs.workshop_id
        WHERE gs.institution_id = @organisation_id OR w.institution_id = @organisation_id
)
SELECT s.id, gs.id AS link_id
FROM share s LEFT JOIN game_share gs ON gs.api_key_share_id = s.id;

-- name: V2DeleteOrganisationInvites :exec
-- After its workshops, shares and links have ended: its invites, to it or
-- to its workshops.
DELETE FROM user_role_invite i
WHERE i.institution_id = @organisation_id
   OR i.workshop_id IN (SELECT w.id FROM workshop w WHERE w.institution_id = @organisation_id);

-- name: V2DeleteOrganisationRoles :exec
DELETE FROM user_role WHERE institution_id = @organisation_id;

-- name: V2HardDeleteOrganisationWorkshops :exec
DELETE FROM workshop WHERE institution_id = @organisation_id;

-- name: V2HardDeleteOrganisation :exec
DELETE FROM institution WHERE id = @organisation_id;

-- Carried over from v1 for the core's system jobs (#344).

-- name: V2SeedOrganisation :one
INSERT INTO institution (
  id, created_by,
  created_at, modified_by, modified_at,
  name
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6
)
RETURNING *;
