-- game_share queries

-- name: CreateGameShare :one
INSERT INTO game_share (game_id, token, api_key_share_id, institution_id, workshop_id, remaining, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetGameShareByToken :one
SELECT * FROM game_share WHERE token = $1;

-- name: GetGameShareByID :one
SELECT * FROM game_share WHERE id = $1;

-- name: GetGameSharesByGameID :many
SELECT * FROM game_share WHERE game_id = $1 ORDER BY created_at;

-- name: GetGameSharesByGameIDAndWorkshop :many
SELECT * FROM game_share WHERE game_id = $1 AND workshop_id = $2 ORDER BY created_at;

-- name: GetGameSharesByGameIDAndCreator :many
-- Personal context: only non-workshop shares (workshop shares belong to their workshop context)
SELECT * FROM game_share WHERE game_id = $1 AND created_by = $2 AND workshop_id IS NULL ORDER BY created_at;

-- name: GetGameSharesByGameIDAndInstitution :many
-- Org shares: shares belonging to an institution (excludes workshop shares which have their own context)
SELECT * FROM game_share WHERE game_id = $1 AND institution_id = $2 AND workshop_id IS NULL ORDER BY created_at;

-- name: GetWorkshopGameShare :one
-- Find existing workshop share for a game (reuse instead of creating duplicates)
SELECT * FROM game_share WHERE game_id = $1 AND workshop_id = $2;

-- name: UpdateGameShareRemaining :one
UPDATE game_share SET remaining = $2 WHERE id = $1 RETURNING *;

-- name: DeleteGameShare :exec
DELETE FROM game_share WHERE id = $1;

-- name: DeleteGameSharesByGameID :exec
DELETE FROM game_share WHERE game_id = $1;

-- name: GetGameShareIDsByApiKeyShareID :many
-- Find game_share IDs that reference a specific api_key_share (for guest cleanup before deletion)
SELECT id FROM game_share WHERE api_key_share_id = $1;

-- name: DeleteGameSharesByApiKeyShareID :exec
-- Cascade cleanup: when an api_key_share is deleted, remove all game_shares referencing it
DELETE FROM game_share WHERE api_key_share_id = $1;

-- name: GetGameShareIDsByApiKeyID :many
-- Find all game_share IDs that reference any api_key_share belonging to a given api_key
SELECT gs.id FROM game_share gs
JOIN api_key_share aks ON aks.id = gs.api_key_share_id
WHERE aks.api_key_id = $1;

-- name: DecrementGameShareRemaining :one
-- Atomically decrements the remaining counter. Returns the share if successful.
-- Succeeds when: remaining is NULL (unlimited) or remaining > 0.
UPDATE game_share SET remaining = CASE
  WHEN remaining IS NULL THEN NULL
  ELSE remaining - 1
END
WHERE id = $1 AND (remaining IS NULL OR remaining > 0) RETURNING *;

-- name: CountGuestUsersByShareID :one
SELECT COUNT(*)::int AS count FROM app_user WHERE private_share_id = $1;

-- Guest data cleanup by share ID

-- name: DeleteGuestSessionMessagesByShareID :exec
DELETE FROM game_session_message WHERE game_session_id IN (
  SELECT gs.id FROM game_session gs
  JOIN app_user u ON u.id = gs.user_id
  WHERE u.private_share_id = $1
);

-- name: DeleteGuestSessionsByShareID :exec
DELETE FROM game_session WHERE user_id IN (
  SELECT id FROM app_user WHERE private_share_id = $1
);

-- name: DeleteGuestUsersByShareID :exec
DELETE FROM app_user WHERE private_share_id = $1;
