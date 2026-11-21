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
SELECT * FROM institution WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateWorkshop :one
INSERT INTO workshop (
  id, created_by,
  created_at, modified_by, modified_at,
  name, institution_id, active, public, default_api_key_share_id,
  ai_quality_tier, prompt_constraints, show_public_games,
  show_other_participants_games, is_paused,
  allow_game_sharing, public_slug
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8, $9, $10,
  $11, $12, $13,
  $14, $15,
  $16, $17
)
RETURNING *;

-- name: GetWorkshopByID :one
SELECT * FROM workshop WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkshops :many
SELECT * FROM workshop WHERE deleted_at IS NULL ORDER BY name;

-- name: UpdateWorkshop :one
UPDATE workshop SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  name = $6,
  institution_id = $7,
  active = $8,
  public = $9,
  default_api_key_share_id = $10,
  ai_quality_tier = $11,
  prompt_constraints = $12,
  show_public_games = $13,
  show_other_participants_games = $14,
  is_paused = $15,
  allow_game_sharing = $16,
  public_slug = $17,
  public_description = $18,
  public_links = $19
WHERE id = $1
RETURNING *;

-- name: DeleteWorkshop :exec
UPDATE workshop SET deleted_at = now() WHERE id = $1;

