-- game_share queries

-- name: GetGameSharesByGameID :many
SELECT * FROM game_share WHERE game_id = $1 ORDER BY created_at;

