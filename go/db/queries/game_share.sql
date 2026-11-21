-- game_share queries

-- name: GetGameShareByToken :one
SELECT * FROM game_share WHERE token = $1;

-- name: GetGameSharesByGameID :many
SELECT * FROM game_share WHERE game_id = $1 ORDER BY created_at;

-- name: DeleteGameSharesByGameID :exec
DELETE FROM game_share WHERE game_id = $1;

