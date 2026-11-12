-- institution registration: applications, consent log, details, admin lock ---

-- name: CreateInstitutionApplication :one
INSERT INTO institution_application (
  id, user_id, kind,
  person_name, person_function, email,
  institution_name, institution_operator, street, postal_code, city,
  prompt_constraint, declaration_version, declaration_hash,
  token_hash, expires_at
) VALUES (
  $1, $2, $3,
  $4, $5, $6,
  $7, $8, $9, $10, $11,
  $12, $13, $14,
  $15, $16
)
RETURNING *;

-- name: GetInstitutionApplicationByTokenHash :one
SELECT * FROM institution_application WHERE token_hash = $1;

-- name: GetPendingInstitutionApplicationByUser :one
SELECT * FROM institution_application
WHERE user_id = $1 AND status = 'pending'
ORDER BY created_at DESC
LIMIT 1;

-- name: ReplaceInstitutionApplicationToken :exec
UPDATE institution_application
SET token_hash = $2, expires_at = $3, modified_at = now()
WHERE id = $1 AND status = 'pending';

-- name: ConfirmInstitutionApplication :exec
UPDATE institution_application
SET status = 'confirmed', confirmed_at = now(), institution_id = $2, modified_at = now()
WHERE id = $1 AND status = 'pending';

-- name: ExpireInstitutionApplicationsForUser :exec
UPDATE institution_application
SET status = 'expired', modified_at = now()
WHERE user_id = $1 AND status = 'pending';

-- name: ExpireOverdueInstitutionApplications :execrows
UPDATE institution_application
SET status = 'expired', modified_at = now()
WHERE status = 'pending' AND expires_at < now();

-- name: AnonymizeOldUnconfirmedInstitutionApplications :execrows
-- Removes the personal data of never-confirmed applications; institution data stays visible.
UPDATE institution_application
SET user_id = NULL, person_name = NULL, person_function = NULL, email = NULL,
    anonymized_at = now(), modified_at = now()
WHERE status <> 'confirmed' AND anonymized_at IS NULL AND created_at < $1;

-- name: ListInstitutionApplications :many
SELECT * FROM institution_application
ORDER BY created_at DESC
LIMIT $1;

-- name: CreateInstitutionConsent :one
INSERT INTO institution_consent (
  id, institution_id, user_id, application_id,
  email, person_name, person_function,
  institution_name, institution_operator, street, postal_code, city,
  declaration_version, declaration_hash
) VALUES (
  $1, $2, $3, $4,
  $5, $6, $7,
  $8, $9, $10, $11, $12,
  $13, $14
)
RETURNING *;

-- name: GetLatestInstitutionConsent :one
SELECT * FROM institution_consent
WHERE institution_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListInstitutionConsents :many
SELECT * FROM institution_consent
WHERE institution_id = $1
ORDER BY created_at DESC;

-- name: UpdateInstitutionDetails :exec
UPDATE institution
SET operator = $2, street = $3, postal_code = $4, city = $5, modified_at = now()
WHERE id = $1;

-- name: SetInstitutionDisabled :exec
UPDATE institution
SET disabled_at = $2, disabled_reason = $3, modified_at = now()
WHERE id = $1;

-- name: InstitutionNameExists :one
-- Case-insensitive and including deleted organisations (the name column is UNIQUE over all rows).
SELECT EXISTS (SELECT 1 FROM institution WHERE lower(name) = lower(sqlc.arg(name)::text));

-- name: GetActiveUserIDsByEmail :many
SELECT id FROM app_user WHERE lower(email) = lower(sqlc.arg(email)::text) AND deleted_at IS NULL;
