
-- api_key_share -------------------------------------------------------

-- name: CreateApiKeyShare :one
INSERT INTO api_key_share (
  id, created_by, created_at, modified_by, modified_at,
  api_key_id, user_id, workshop_id, institution_id, allow_public_sponsored_plays
) VALUES (
  gen_random_uuid(), $1, $2, $3, $4,
  $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetApiKeyShareByID :one
SELECT
  s.id,
  s.created_by,
  s.created_at,
  s.modified_by,
  s.modified_at,
  s.api_key_id,
  s.user_id,
  s.workshop_id,
  s.institution_id,
  s.allow_public_sponsored_plays,
  k.id AS key_id,
  k.user_id AS key_owner_id,
  k.name AS key_name,
  k.platform AS key_platform,
  k.key AS key_key,
  k.is_default AS key_is_default,
  k.last_usage_success AS key_last_usage_success,
  o.name AS key_owner_name
FROM api_key_share s
JOIN api_key k ON k.id = s.api_key_id
JOIN app_user o ON o.id = k.user_id
WHERE s.id = $1;

-- name: GetApiKeySharesByApiKeyID :many
SELECT
  s.id,
  s.created_by,
  s.created_at,
  s.modified_by,
  s.modified_at,
  s.api_key_id,
  s.user_id,
  s.workshop_id,
  s.institution_id,
  s.allow_public_sponsored_plays,
  u.name AS user_name,
  w.name AS workshop_name,
  i.name AS institution_name
FROM api_key_share s
LEFT JOIN app_user u ON u.id = s.user_id
LEFT JOIN workshop w ON w.id = s.workshop_id
LEFT JOIN institution i ON i.id = s.institution_id
WHERE s.api_key_id = $1;

-- name: GetApiKeySharesByUserID :many
SELECT
  s.id,
  s.created_by,
  s.created_at,
  s.modified_by,
  s.modified_at,
  s.api_key_id,
  s.user_id,
  s.workshop_id,
  s.institution_id,
  s.allow_public_sponsored_plays,
  k.name AS api_key_name,
  k.platform AS api_key_platform,
  k.key AS api_key_key,
  k.is_default AS api_key_is_default,
  k.last_usage_success AS api_key_last_usage_success,
  k.user_id AS owner_id,
  owner.name AS owner_name
FROM api_key_share s
JOIN api_key k ON k.id = s.api_key_id
JOIN app_user owner ON owner.id = k.user_id
WHERE s.user_id = $1;

-- name: DeleteApiKeyShare :exec
DELETE FROM api_key_share WHERE id = $1;

-- name: DeleteApiKeySharesByApiKeyID :exec
DELETE FROM api_key_share WHERE api_key_id = $1;

-- name: GetApiKeySharesByInstitutionID :many
SELECT
  s.id,
  s.created_by,
  s.created_at,
  s.modified_by,
  s.modified_at,
  s.api_key_id,
  s.user_id,
  s.workshop_id,
  s.institution_id,
  s.allow_public_sponsored_plays,
  k.name AS api_key_name,
  k.platform AS api_key_platform,
  k.user_id AS owner_id,
  owner.name AS owner_name
FROM api_key_share s
JOIN api_key k ON k.id = s.api_key_id
JOIN app_user owner ON owner.id = k.user_id
WHERE s.institution_id = $1;

-- name: ClearUserDefaultApiKeyShareByApiKeyID :exec
UPDATE app_user
SET default_api_key_share_id = NULL, modified_at = now()
WHERE default_api_key_share_id IN (
  SELECT id FROM api_key_share WHERE api_key_id = $1
);

-- name: ClearSessionApiKeyID :exec
UPDATE game_session SET api_key_id = NULL, modified_at = now() WHERE api_key_id = $1;

-- name: ClearGameSponsoredApiKey :exec
UPDATE game
SET
  public_sponsored_api_key_id = CASE WHEN public_sponsored_api_key_id = $1 THEN NULL ELSE public_sponsored_api_key_id END,
  private_sponsored_api_key_id = CASE WHEN private_sponsored_api_key_id = $1 THEN NULL ELSE private_sponsored_api_key_id END,
  modified_at = now()
WHERE public_sponsored_api_key_id = $1 OR private_sponsored_api_key_id = $1;

-- name: DeleteApiKeySharesByOwnerForInstitution :exec
-- Delete all API key shares owned by a user that target a specific institution
DELETE FROM api_key_share s
WHERE s.api_key_id IN (SELECT k.id FROM api_key k WHERE k.user_id = $1)
  AND s.institution_id = $2;

-- name: DeleteApiKeySharesByOwnerForInstitutionWorkshops :exec
-- Delete all API key shares owned by a user that target any workshop in a specific institution
DELETE FROM api_key_share s
WHERE s.api_key_id IN (SELECT k.id FROM api_key k WHERE k.user_id = $1)
  AND s.workshop_id IN (SELECT w.id FROM workshop w WHERE w.institution_id = $2);

-- name: GetWorkshopIDsByInstitution :many
SELECT id FROM workshop WHERE institution_id = $1 AND deleted_at IS NULL;
