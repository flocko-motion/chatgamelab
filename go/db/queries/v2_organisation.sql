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
INSERT INTO institution (
  id, created_by, modified_by, name,
  operator, street, postal_code, city, prompt_constraints
) VALUES (
  @id, @creator::uuid, @creator::uuid, @name,
  @operator::text, @street::text, @postal_code::text, @city::text, @prompt_constraints::text
);

-- name: V2Organisation :one
-- What a head confirms with the declaration: the organisation's name,
-- details and youth-protection restriction.
SELECT id, name, operator, street, postal_code, city, prompt_constraints
FROM institution
WHERE id = @id AND deleted_at IS NULL;

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
