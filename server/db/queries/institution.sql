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
