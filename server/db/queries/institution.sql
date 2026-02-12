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

-- name: ListInstitutions :many
SELECT * FROM institution WHERE deleted_at IS NULL ORDER BY name;

-- name: UpdateInstitution :one
UPDATE institution SET
  created_by = $2,
  created_at = $3,
  modified_by = $4,
  modified_at = $5,
  name = $6
WHERE id = $1
RETURNING *;

-- name: SetInstitutionFreeUseApiKeyShare :exec
UPDATE institution
SET free_use_api_key_share_id = $2, modified_at = now()
WHERE id = $1;

-- name: UpdateInstitutionFreeUseAiQualityTier :exec
UPDATE institution
SET free_use_ai_quality_tier = $2, modified_at = now()
WHERE id = $1;

-- name: SoftDeleteInstitution :exec
UPDATE institution SET deleted_at = now() WHERE id = $1;

-- name: HardDeleteInstitution :exec
DELETE FROM institution WHERE id = $1;

-- name: GetParticipantUserIDsByInstitution :many
SELECT DISTINCT u.id
FROM app_user u
JOIN user_role r ON u.id = r.user_id
WHERE r.institution_id = $1
  AND r.role = 'participant'
  AND u.deleted_at IS NULL;

-- name: GetNonParticipantUserIDsByInstitution :many
SELECT DISTINCT u.id
FROM app_user u
JOIN user_role r ON u.id = r.user_id
WHERE r.institution_id = $1
  AND r.role != 'participant'
  AND u.deleted_at IS NULL;

-- name: DeleteInvitesByInstitution :exec
DELETE FROM user_role_invite WHERE institution_id = $1;

-- name: DeleteApiKeySharesByInstitution :exec
DELETE FROM api_key_share WHERE institution_id = $1;

-- name: DeleteUserRolesByInstitution :exec
DELETE FROM user_role WHERE institution_id = $1;

-- name: HardDeleteWorkshopsByInstitution :exec
DELETE FROM workshop WHERE institution_id = $1;

-- name: ClearInstitutionFreeUseApiKeyShare :exec
UPDATE institution SET free_use_api_key_share_id = NULL WHERE id = $1;

-- name: GetInstitutionMembers :many
SELECT u.id, u.name, u.email, r.role
FROM app_user u
JOIN user_role r ON u.id = r.user_id
WHERE r.institution_id = $1
  AND u.deleted_at IS NULL
  AND r.role IN ('individual', 'staff', 'head')
ORDER BY r.role, u.name;


-- workshop -------------------------------------------------------------

-- name: CreateWorkshop :one
INSERT INTO workshop (
  id, created_by,
  created_at, modified_by, modified_at,
  name, institution_id, active, public, default_api_key_share_id,
  ai_quality_tier, show_public_games, show_other_participants_games,
  design_editing_enabled, is_paused
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8, $9, $10,
  $11, $12, $13,
  $14, $15
)
RETURNING *;

-- name: GetWorkshopByID :one
SELECT * FROM workshop WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkshops :many
SELECT * FROM workshop WHERE deleted_at IS NULL ORDER BY name;

-- name: ListWorkshopsByInstitution :many
SELECT * FROM workshop WHERE institution_id = $1 AND deleted_at IS NULL ORDER BY name;

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
  show_public_games = $12,
  show_other_participants_games = $13,
  design_editing_enabled = $14,
  is_paused = $15
WHERE id = $1
RETURNING *;

-- name: SetWorkshopDefaultApiKey :one
UPDATE workshop SET
  modified_by = $2,
  modified_at = now(),
  default_api_key_share_id = $3
WHERE id = $1
RETURNING *;

-- name: DeleteWorkshop :exec
UPDATE workshop SET deleted_at = now() WHERE id = $1;


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

-- name: DeleteWorkshopParticipantsByWorkshopID :exec
DELETE FROM workshop_participant WHERE workshop_id = $1;
