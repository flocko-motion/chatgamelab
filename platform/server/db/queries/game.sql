-- Basic CRUD queries for core entities
-- Exactly four per table: create, read by id, update by id, delete by id.


-- game -----------------------------------------------------------------

-- name: CreateGame :one
INSERT INTO game (
  id, created_by,
  created_at, modified_by, modified_at,
  name, description, icon,
  workshop_id,
  public, public_sponsored_api_key_share_id,
  system_message_scenario, system_message_game_start,
  image_style, css, status_fields,
  theme,
  first_message, first_status, first_image,
  originally_created_by, play_count, clone_count
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8,
  $9,
  $10, $11,
  $12, $13,
  $14, $15, $16,
  $17,
  $18, $19, $20,
  $21, $22, $23
)
RETURNING *;

-- name: GetGameByID :one
SELECT * FROM game WHERE id = $1;

-- name: GetGamesVisibleToUser :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY created_at DESC;

-- name: GetGamesVisibleToUserSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY LOWER(name) ASC;

-- name: GetGamesVisibleToUserSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY LOWER(name) DESC;

-- name: GetGamesVisibleToUserSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY created_at ASC;

-- name: GetGamesVisibleToUserSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY modified_at DESC;

-- name: GetGamesVisibleToUserSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY modified_at ASC;

-- name: SearchGamesVisibleToUser :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY created_at DESC;

-- name: SearchGamesVisibleToUserSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchGamesVisibleToUserSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchGamesVisibleToUserSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY created_at ASC;

-- name: SearchGamesVisibleToUserSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY modified_at DESC;

-- name: SearchGamesVisibleToUserSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY modified_at ASC;

-- name: GetGameIDsVisibleToUser :many
SELECT id FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2);

-- Own games (created by user) queries
-- name: GetOwnGames :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY created_at DESC;

-- name: GetOwnGamesSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY LOWER(name) ASC;

-- name: GetOwnGamesSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY LOWER(name) DESC;

-- name: GetOwnGamesSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY created_at ASC;

-- name: GetOwnGamesSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY modified_at DESC;

-- name: GetOwnGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY modified_at ASC;

-- name: SearchOwnGames :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY created_at DESC;

-- name: SearchOwnGamesSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchOwnGamesSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchOwnGamesSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY created_at ASC;

-- name: SearchOwnGamesSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY modified_at DESC;

-- name: SearchOwnGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY modified_at ASC;

-- name: GetOwnGamesSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY play_count DESC;

-- name: GetOwnGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY play_count ASC;

-- name: GetOwnGamesSortedByVisibility :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY public DESC, modified_at DESC;

-- name: GetOwnGamesSortedByVisibilityAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 ORDER BY public ASC, modified_at DESC;

-- name: SearchOwnGamesSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY play_count DESC;

-- name: SearchOwnGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY play_count ASC;

-- name: SearchOwnGamesSortedByVisibility :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY public DESC, modified_at DESC;

-- name: SearchOwnGamesSortedByVisibilityAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY public ASC, modified_at DESC;

-- name: GetPublicGames :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY created_at DESC;

-- name: GetPublicGamesSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY LOWER(name) ASC;

-- name: GetPublicGamesSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY LOWER(name) DESC;

-- name: GetPublicGamesSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY created_at ASC;

-- name: GetPublicGamesSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY modified_at DESC;

-- name: GetPublicGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY modified_at ASC;

-- name: SearchPublicGames :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY created_at DESC;

-- name: SearchPublicGamesSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchPublicGamesSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchPublicGamesSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY created_at ASC;

-- name: SearchPublicGamesSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY modified_at DESC;

-- name: SearchPublicGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY modified_at ASC;

-- name: GetPublicGamesSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY play_count DESC;

-- name: GetPublicGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true ORDER BY play_count ASC;

-- All games queries (admin only)
-- name: GetAllGames :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY created_at DESC;

-- name: GetAllGamesSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY LOWER(name) ASC;

-- name: GetAllGamesSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY LOWER(name) DESC;

-- name: GetAllGamesSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY created_at ASC;

-- name: GetAllGamesSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY modified_at DESC;

-- name: GetAllGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY modified_at ASC;

-- name: SearchAllGames :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY created_at DESC;

-- name: SearchAllGamesSortedByName :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchAllGamesSortedByNameDesc :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchAllGamesSortedByCreatedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY created_at ASC;

-- name: SearchAllGamesSortedByModifiedAt :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY modified_at DESC;

-- name: SearchAllGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY modified_at ASC;

-- name: GetAllGamesSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY play_count DESC;

-- name: GetAllGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL ORDER BY play_count ASC;

-- name: SearchAllGamesSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY play_count DESC;

-- name: SearchAllGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY play_count ASC;

-- name: SearchPublicGamesSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY play_count DESC;

-- name: SearchPublicGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY play_count ASC;

-- Games visible to user with additional sort options
-- name: GetGamesVisibleToUserSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY play_count DESC;

-- name: GetGamesVisibleToUserSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) ORDER BY play_count ASC;

-- name: SearchGamesVisibleToUserSortedByPlayCount :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY play_count DESC;

-- name: SearchGamesVisibleToUserSortedByPlayCountAsc :many
SELECT * FROM game WHERE deleted_at IS NULL AND (created_by = $1 OR public = true OR workshop_id = $2) AND LOWER(name) LIKE LOWER('%' || $3 || '%') ORDER BY play_count ASC;

-- Creator sorting requires joining with user table
-- name: GetGamesVisibleToUserSortedByCreator :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE g.deleted_at IS NULL AND (g.created_by = $1 OR g.public = true) ORDER BY LOWER(COALESCE(u.name, '')) ASC;

-- name: GetGamesVisibleToUserSortedByCreatorDesc :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE g.deleted_at IS NULL AND (g.created_by = $1 OR g.public = true) ORDER BY LOWER(COALESCE(u.name, '')) DESC;

-- name: SearchGamesVisibleToUserSortedByCreator :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE g.deleted_at IS NULL AND (g.created_by = $1 OR g.public = true) AND LOWER(g.name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(COALESCE(u.name, '')) ASC;

-- name: SearchGamesVisibleToUserSortedByCreatorDesc :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE g.deleted_at IS NULL AND (g.created_by = $1 OR g.public = true) AND LOWER(g.name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(COALESCE(u.name, '')) DESC;

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
  public_sponsored_api_key_share_id = $10,
  system_message_scenario = $11,
  system_message_game_start = $12,
  image_style = $13,
  css = $14,
  status_fields = $15,
  theme = $16,
  first_message = $17,
  first_status = $18,
  first_image = $19,
  originally_created_by = $20
WHERE id = $1
RETURNING *;

-- name: IncrementGamePlayCount :exec
UPDATE game SET play_count = play_count + 1 WHERE id = $1;

-- name: IncrementGameCloneCount :exec
UPDATE game SET clone_count = clone_count + 1 WHERE id = $1;

-- name: SoftDeleteGame :exec
UPDATE game SET deleted_at = now(), modified_at = now() WHERE id = $1;

-- name: HardDeleteGame :exec
DELETE FROM game WHERE id = $1;

-- User deletion: game cleanup queries

-- name: GetGameIDsByCreator :many
SELECT id FROM game WHERE created_by = $1;

-- name: DeleteGameTagsByGameID :exec
DELETE FROM game_tag WHERE game_id = $1;

-- name: DeleteFavouritesByGameID :exec
DELETE FROM user_favourite_game WHERE game_id = $1;

-- name: HardDeleteGamesByCreator :exec
DELETE FROM game WHERE created_by = $1;

-- name: UnlinkGamesFromWorkshop :exec
UPDATE game SET workshop_id = NULL WHERE workshop_id = $1;


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

-- name: GetPublicGamesByWorkshop :many
-- Games shown on a workshop's public page.
SELECT * FROM game WHERE workshop_id = $1 AND public = true AND deleted_at IS NULL ORDER BY LOWER(name) ASC;
