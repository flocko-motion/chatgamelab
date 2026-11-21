
-- api_key_share -------------------------------------------------------

-- name: CreateApiKeyShare :one
INSERT INTO api_key_share (
  id, created_by, created_at, modified_by, modified_at,
  api_key_id, user_id, workshop_id, institution_id, game_id
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
  s.game_id,

  k.id AS key_id,
  k.user_id AS key_owner_id,
  k.name AS key_name,
  k.platform AS key_platform,
  k.key AS key_key,
  k.is_default AS key_is_default,
  k.last_usage_success AS key_last_usage_success,
  k.last_error_code AS key_last_error_code,
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
  s.game_id,

  u.name AS user_name,
  w.name AS workshop_name,
  i.name AS institution_name,
  g.name AS game_name
FROM api_key_share s
LEFT JOIN app_user u ON u.id = s.user_id
LEFT JOIN workshop w ON w.id = s.workshop_id
LEFT JOIN institution i ON i.id = s.institution_id
LEFT JOIN game g ON g.id = s.game_id
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
  s.game_id,

  k.name AS api_key_name,
  k.platform AS api_key_platform,
  k.key AS api_key_key,
  k.is_default AS api_key_is_default,
  k.last_usage_success AS api_key_last_usage_success,
  k.last_error_code AS api_key_last_error_code,
  k.user_id AS owner_id,
  owner.name AS owner_name
FROM api_key_share s
JOIN api_key k ON k.id = s.api_key_id
JOIN app_user owner ON owner.id = k.user_id
WHERE s.user_id = $1;

-- name: DeleteApiKeySharesByApiKeyID :exec
DELETE FROM api_key_share WHERE api_key_id = $1;

-- name: ClearUserDefaultApiKeyShareByApiKeyID :exec
UPDATE app_user
SET default_api_key_share_id = NULL, modified_at = now()
WHERE default_api_key_share_id IN (
  SELECT id FROM api_key_share WHERE api_key_id = $1
);

-- name: ClearWorkshopDefaultApiKeyShareByApiKeyID :exec
-- Clear workshop default API key when an API key is deleted (find shares for that key)
UPDATE workshop
SET default_api_key_share_id = NULL, modified_at = now()
WHERE default_api_key_share_id IN (
  SELECT id FROM api_key_share WHERE api_key_id = $1
);

-- name: ClearGameSponsoredApiKeyByApiKeyID :exec
-- Clear game sponsoring when an API key is deleted (find shares for that key)
UPDATE game g
SET
  public_sponsored_api_key_share_id = NULL,
  modified_at = now()
WHERE g.public_sponsored_api_key_share_id IN (SELECT s.id FROM api_key_share s WHERE s.api_key_id = $1);

-- name: DeleteApiKeySharesByGameID :exec
DELETE FROM api_key_share WHERE game_id = $1;

