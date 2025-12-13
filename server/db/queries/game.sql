-- Basic CRUD queries for core entities
-- Exactly four per table: create, read by id, update by id, delete by id.


-- game -----------------------------------------------------------------

-- name: CreateGame :one
INSERT INTO game (
  id, created_by,
  created_at, modified_by, modified_at,
  name, description, icon,
  public, public_sponsored_api_key_id,
  private_share_hash, private_sponsored_api_key_id,
  system_message_scenario, system_message_game_start,
  image_style, css, status_fields,
  first_message, first_status, first_image
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8,
  $9, $10,
  $11, $12,
  $13, $14,
  $15, $16, $17,
  $18, $19, $20
)
RETURNING *;

-- name: GetGameByID :one
SELECT * FROM game WHERE id = $1;

-- name: GetGamesVisibleToUser :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY created_at DESC;

-- name: GetGameIDsVisibleToUser :many
SELECT id FROM game WHERE created_by = $1 OR public = true;

-- name: GetPublicGames :many
SELECT * FROM game WHERE public = true ORDER BY created_at DESC;

-- name: GetGameByPrivateShareHash :one
SELECT * FROM game WHERE private_share_hash = $1;

-- name: GetGameTagsByGameID :many
SELECT * FROM game_tag WHERE game_id = $1;

-- name: UpdateGame :one
UPDATE game SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  name = $6,
  description = $7,
  icon = $8,
  public = $9,
  public_sponsored_api_key_id = $10,
  private_share_hash = $11,
  private_sponsored_api_key_id = $12,
  system_message_scenario = $13,
  system_message_game_start = $14,
  image_style = $15,
  css = $16,
  status_fields = $17,
  first_message = $18,
  first_status = $19,
  first_image = $20
WHERE id = $1
RETURNING *;

-- name: DeleteGame :exec
DELETE FROM game WHERE id = $1;


-- game_tag -------------------------------------------------------------

-- name: CreateGameTag :one
INSERT INTO game_tag (
  id, created_by,
  created_at, modified_by, modified_at,
  game_id, tag
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7
)
RETURNING *;

-- name: GetGameTagByID :one
SELECT * FROM game_tag WHERE id = $1;

-- name: UpdateGameTag :one
UPDATE game_tag SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  game_id = $6,
  tag = $7
WHERE id = $1
RETURNING *;

-- name: DeleteGameTag :exec
DELETE FROM game_tag WHERE id = $1;


-- game_session ---------------------------------------------------------

-- name: CreateGameSession :one
INSERT INTO game_session (
  id, created_by,
  created_at, modified_by, modified_at,
  game_id, user_id, api_key_id,
  ai_platform, ai_model, ai_session,
  image_style, status_fields
) VALUES (
  gen_random_uuid(), $1,
  $2, $3, $4,
  $5, $6, $7,
  $8, $9, $10,
  $11, $12
)
RETURNING *;

-- name: GetGameSessionByID :one
SELECT * FROM game_session WHERE id = $1;

-- name: GetGameSessionsByGameID :many
SELECT * FROM game_session WHERE game_id = $1 ORDER BY created_at DESC;

-- name: UpdateGameSession :one
UPDATE game_session SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  game_id = $6,
  user_id = $7,
  api_key_id = $8,
  ai_platform = $9,
  ai_model = $10,
  ai_session = $11,
  image_style = $12,
  status_fields = $13
WHERE id = $1
RETURNING *;

-- name: DeleteGameSession :exec
DELETE FROM game_session WHERE id = $1;


-- game_session_message -------------------------------------------------

-- name: CreateGameSessionMessage :one
INSERT INTO game_session_message (
  id, created_by,
  created_at, modified_by, modified_at,
  game_session_id, seq,
  type, message,
  status, image_prompt, image
) VALUES (
  gen_random_uuid(), $1,
  $2, $3, $4,
  $5, (SELECT COALESCE(MAX(seq), 0) + 1 FROM game_session_message WHERE game_session_id = $5),
  $6, $7,
  $8, $9, $10
)
RETURNING *;

-- name: GetGameSessionMessageByID :one
SELECT * FROM game_session_message WHERE id = $1;

-- name: UpdateGameSessionMessage :one
UPDATE game_session_message SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  game_session_id = $6,
  type = $7,
  message = $8,
  status = $9,
  image_prompt = $10,
  image = $11
WHERE id = $1
RETURNING *;

-- name: DeleteGameSessionMessage :exec
DELETE FROM game_session_message WHERE id = $1;

-- name: UpdateGameSessionAiSession :one
UPDATE game_session SET
  ai_session = $2,
  modified_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateGameSessionMessageImage :one
UPDATE game_session_message SET
  image = $2,
  modified_at = now()
WHERE id = $1
RETURNING *;
