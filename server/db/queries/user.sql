
-- app_user -------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO app_user (id, name, email, auth0_id)
VALUES (gen_random_uuid(), $1, $2, $3)
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
