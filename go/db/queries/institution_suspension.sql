-- an institution's suspension as v1's share links and public pages see it.
-- Registering, consenting and suspending are the core's (v2_organisation.sql).

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

