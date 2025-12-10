

-- api_key_share_user ---------------------------------------------------

-- name: CreateApiKeyShareUser :one
INSERT INTO api_key_share_user (
  id, created_by,
  created_at, modified_by, modified_at,
  api_key_id, user_id, allow_public_sponsored_plays
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8
)
RETURNING *;

-- name: GetApiKeyShareUserByID :one
SELECT * FROM api_key_share_user WHERE id = $1;

-- name: UpdateApiKeyShareUser :one
UPDATE api_key_share_user SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  api_key_id = $6,
  user_id = $7,
  allow_public_sponsored_plays = $8
WHERE id = $1
RETURNING *;

-- name: DeleteApiKeyShareUser :exec
DELETE FROM api_key_share_user WHERE id = $1;


-- api_key_share_workshop ----------------------------------------------

-- name: CreateApiKeyShareWorkshop :one
INSERT INTO api_key_share_workshop (
  id, created_by,
  created_at, modified_by, modified_at,
  api_key_id, workshop_id, allow_public_sponsored_plays
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8
)
RETURNING *;

-- name: GetApiKeyShareWorkshopByID :one
SELECT * FROM api_key_share_workshop WHERE id = $1;

-- name: UpdateApiKeyShareWorkshop :one
UPDATE api_key_share_workshop SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  api_key_id = $6,
  workshop_id = $7,
  allow_public_sponsored_plays = $8
WHERE id = $1
RETURNING *;

-- name: DeleteApiKeyShareWorkshop :exec
DELETE FROM api_key_share_workshop WHERE id = $1;

