-- user_favourite_game -----------------------------------------------------
-- name: AddFavouriteGame :one
INSERT INTO user_favourite_game (
        id,
        created_by,
        created_at,
        modified_by,
        modified_at,
        user_id,
        game_id
    )
VALUES (gen_random_uuid(), $1, now(), $1, now(), $1, $2) ON CONFLICT (user_id, game_id) DO NOTHING
RETURNING id;
-- name: RemoveFavouriteGame :exec
DELETE FROM user_favourite_game
WHERE user_id = $1
    AND game_id = $2;
-- name: GetFavouriteGamesByUserID :many
SELECT g.*
FROM game g
    JOIN user_favourite_game f ON f.game_id = g.id
WHERE f.user_id = $1
ORDER BY f.created_at DESC;
-- name: IsFavouriteGame :one
SELECT EXISTS(
        SELECT 1
        FROM user_favourite_game
        WHERE user_id = $1
            AND game_id = $2
    ) AS is_favourite;
