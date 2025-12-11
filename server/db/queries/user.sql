
-- app_user -------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id;

-- name: CreateUserWithID :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: GetUserIDByAuth0ID :one
SELECT id FROM app_user WHERE auth0_id = $1;

-- name: GetUserByID :one
SELECT * FROM app_user WHERE id = $1;

-- name: GetUserDetailsByID :one
SELECT
  u.id,
  u.created_by,
  u.created_at,
  u.modified_by,
  u.modified_at,
  u.name,
  u.email,
  u.deleted_at,
  u.auth0_id,
  r.id           AS role_id,
  r.role         AS role,
  r.institution_id,
  i.name         AS institution_name
FROM app_user u
LEFT JOIN LATERAL (
  SELECT ur.*
  FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY ur.created_at DESC
  LIMIT 1
) r ON TRUE
LEFT JOIN institution i
  ON i.id = r.institution_id
WHERE u.id = $1;

-- name: GetUserApiKeys :many
SELECT
  id,
  created_by,
  created_at,
  modified_by,
  modified_at,
  user_id,
  platform,
  key
FROM api_key
WHERE user_id = $1;

-- name: UpdateUser :exec
UPDATE app_user SET
  name = $2,
  email = $3,
  modified_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
UPDATE app_user
SET
  deleted_at = now()
WHERE id = $1;

-- user_role -------------------------------------------------------------

-- name: DeleteUserRoles :exec
DELETE FROM user_role WHERE user_id = $1;

-- name: CreateUserRole :one
INSERT INTO user_role (id, user_id, role, institution_id)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id;


-- api_key --------------------------------------------------------------

-- name: CreateApiKey :one
INSERT INTO api_key (
  id, created_by,
  created_at, modified_by, modified_at,
  user_id, platform, key
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8
)
RETURNING *;

-- name: GetApiKeyByID :one
SELECT * FROM api_key WHERE id = $1;

-- name: UpdateApiKey :one
UPDATE api_key SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  user_id = $6,
  platform = $7,
  key = $8
WHERE id = $1
RETURNING *;

-- name: DeleteApiKey :exec
DELETE FROM api_key WHERE id = $1;

-- name: GetApiKeySharesByUserID :many
SELECT
  s.id,
  s.created_by,
  s.created_at,
  s.modified_by,
  s.modified_at,
  s.api_key_id,
  s.user_id,
  s.allow_public_sponsored_plays,
  k.platform AS api_key_platform,
  k.key AS api_key_key
FROM api_key_share_user s
JOIN api_key k ON k.id = s.api_key_id
WHERE s.user_id = $1;

