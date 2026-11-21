-- institutions registering themselves, beside the core: the suspension as
-- share links and public pages see it, an institution's details, and its
-- consent log as v1's institution endpoints show it. Registering, consenting
-- and suspending are the core's (v2_organisation.sql).

-- name: IsInstitutionSuspended :one
-- Whether the institution is suspended: by an admin, or for want of a consent
-- to the declaration in @version and @hash.
SELECT (suspended_at IS NOT NULL OR institution_consent_missing(id, @version::text, @hash::text))::bool AS suspended
FROM institution WHERE id = @id;

-- name: IsWorkshopInstitutionSuspended :one
-- Whether the workshop's institution is suspended (see IsInstitutionSuspended).
SELECT (i.suspended_at IS NOT NULL OR institution_consent_missing(i.id, @version::text, @hash::text))::bool AS suspended
FROM workshop w JOIN institution i ON i.id = w.institution_id
WHERE w.id = @workshop_id;

-- name: UpdateInstitutionDetails :exec
-- Operator and address, as an admin enters them creating an institution.
UPDATE institution
SET operator = $2, street = $3, postal_code = $4, city = $5, modified_at = now()
WHERE id = $1;

-- name: GetLatestInstitutionConsent :many
-- The institution's newest entry in the consent log, if any.
SELECT kind, declaration_version, declaration_hash, created_at
FROM institution_consent
WHERE institution_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetFirstInstitutionConsent :many
-- The institution's first consent, if any: given by its creator, it is how
-- the institution registered itself.
SELECT user_id, created_at, person_name, person_function, email
FROM institution_consent
WHERE institution_id = $1 AND kind = 'consent'
ORDER BY created_at
LIMIT 1;
