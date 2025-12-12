
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
SELECT * FROM api_key_share WHERE id = $1;

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

