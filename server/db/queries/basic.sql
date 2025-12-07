-- Basic CRUD queries for core entities
-- Exactly four per table: create, read by id, update by id, delete by id.

-- institution ----------------------------------------------------------

-- name: CreateInstitution :one
INSERT INTO institution (
  id, created_by,
  created_at, modified_by, modified_at,
  name
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6
)
RETURNING *;

-- name: GetInstitutionByID :one
SELECT * FROM institution WHERE id = $1;

-- name: UpdateInstitution :one
UPDATE institution SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  name = $6
WHERE id = $1
RETURNING *;

-- name: DeleteInstitution :exec
DELETE FROM institution WHERE id = $1;


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


-- workshop -------------------------------------------------------------

-- name: CreateWorkshop :one
INSERT INTO workshop (
  id, created_by,
  created_at, modified_by, modified_at,
  name, institution_id, active, public
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8, $9
)
RETURNING *;

-- name: GetWorkshopByID :one
SELECT * FROM workshop WHERE id = $1;

-- name: UpdateWorkshop :one
UPDATE workshop SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  name = $6,
  institution_id = $7,
  active = $8,
  public = $9
WHERE id = $1
RETURNING *;

-- name: DeleteWorkshop :exec
DELETE FROM workshop WHERE id = $1;


-- workshop_participant -------------------------------------------------

-- name: CreateWorkshopParticipant :one
INSERT INTO workshop_participant (
  id, created_by,
  created_at, modified_by, modified_at,
  workshop_id, name, access_token, active
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8, $9
)
RETURNING *;

-- name: GetWorkshopParticipantByID :one
SELECT * FROM workshop_participant WHERE id = $1;

-- name: UpdateWorkshopParticipant :one
UPDATE workshop_participant SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  workshop_id = $6,
  name = $7,
  access_token = $8,
  active = $9
WHERE id = $1
RETURNING *;

-- name: DeleteWorkshopParticipant :exec
DELETE FROM workshop_participant WHERE id = $1;


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
  model, model_session,
  image_style, status_fields
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8,
  $9, $10,
  $11, $12
)
RETURNING *;

-- name: GetGameSessionByID :one
SELECT * FROM game_session WHERE id = $1;

-- name: UpdateGameSession :one
UPDATE game_session SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  game_id = $6,
  user_id = $7,
  api_key_id = $8,
  model = $9,
  model_session = $10,
  image_style = $11,
  status_fields = $12
WHERE id = $1
RETURNING *;

-- name: DeleteGameSession :exec
DELETE FROM game_session WHERE id = $1;


-- game_session_message -------------------------------------------------

-- name: CreateGameSessionMessage :one
INSERT INTO game_session_message (
  id, created_by,
  created_at, modified_by, modified_at,
  game_session_id,
  type, message,
  status, image_prompt, image
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6,
  $7, $8,
  $9, $10, $11
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
