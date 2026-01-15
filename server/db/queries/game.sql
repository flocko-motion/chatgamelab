-- Basic CRUD queries for core entities
-- Exactly four per table: create, read by id, update by id, delete by id.


-- game -----------------------------------------------------------------

-- name: CreateGame :one
INSERT INTO game (
  id, created_by,
  created_at, modified_by, modified_at,
  name, description, icon,
  workshop_id,
  public, public_sponsored_api_key_id,
  private_share_hash, private_sponsored_api_key_id,
  system_message_scenario, system_message_game_start,
  image_style, css, status_fields,
  first_message, first_status, first_image,
  originally_created_by
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8,
  $9,
  $10, $11,
  $12, $13,
  $14, $15,
  $16, $17, $18,
  $19, $20, $21
)
RETURNING *;

-- name: GetGameByID :one
SELECT * FROM game WHERE id = $1;

-- name: GetGamesVisibleToUser :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY created_at DESC;

-- name: GetGamesVisibleToUserSortedByName :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY LOWER(name) ASC;

-- name: GetGamesVisibleToUserSortedByNameDesc :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY LOWER(name) DESC;

-- name: GetGamesVisibleToUserSortedByCreatedAt :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY created_at ASC;

-- name: GetGamesVisibleToUserSortedByModifiedAt :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY modified_at DESC;

-- name: GetGamesVisibleToUserSortedByModifiedAtAsc :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY modified_at ASC;

-- name: SearchGamesVisibleToUser :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY created_at DESC;

-- name: SearchGamesVisibleToUserSortedByName :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchGamesVisibleToUserSortedByNameDesc :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchGamesVisibleToUserSortedByCreatedAt :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY created_at ASC;

-- name: SearchGamesVisibleToUserSortedByModifiedAt :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY modified_at DESC;

-- name: SearchGamesVisibleToUserSortedByModifiedAtAsc :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY modified_at ASC;

-- name: GetGameIDsVisibleToUser :many
SELECT id FROM game WHERE created_by = $1 OR public = true;

-- Own games (created by user) queries
-- name: GetOwnGames :many
SELECT * FROM game WHERE created_by = $1 ORDER BY created_at DESC;

-- name: GetOwnGamesSortedByName :many
SELECT * FROM game WHERE created_by = $1 ORDER BY LOWER(name) ASC;

-- name: GetOwnGamesSortedByNameDesc :many
SELECT * FROM game WHERE created_by = $1 ORDER BY LOWER(name) DESC;

-- name: GetOwnGamesSortedByCreatedAt :many
SELECT * FROM game WHERE created_by = $1 ORDER BY created_at ASC;

-- name: GetOwnGamesSortedByModifiedAt :many
SELECT * FROM game WHERE created_by = $1 ORDER BY modified_at DESC;

-- name: GetOwnGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE created_by = $1 ORDER BY modified_at ASC;

-- name: SearchOwnGames :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY created_at DESC;

-- name: SearchOwnGamesSortedByName :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchOwnGamesSortedByNameDesc :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchOwnGamesSortedByCreatedAt :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY created_at ASC;

-- name: SearchOwnGamesSortedByModifiedAt :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY modified_at DESC;

-- name: SearchOwnGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY modified_at ASC;

-- name: GetOwnGamesSortedByPlayCount :many
SELECT * FROM game WHERE created_by = $1 ORDER BY play_count DESC;

-- name: GetOwnGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE created_by = $1 ORDER BY play_count ASC;

-- name: GetOwnGamesSortedByVisibility :many
SELECT * FROM game WHERE created_by = $1 ORDER BY public DESC, modified_at DESC;

-- name: GetOwnGamesSortedByVisibilityAsc :many
SELECT * FROM game WHERE created_by = $1 ORDER BY public ASC, modified_at DESC;

-- name: SearchOwnGamesSortedByPlayCount :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY play_count DESC;

-- name: SearchOwnGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY play_count ASC;

-- name: SearchOwnGamesSortedByVisibility :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY public DESC, modified_at DESC;

-- name: SearchOwnGamesSortedByVisibilityAsc :many
SELECT * FROM game WHERE created_by = $1 AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY public ASC, modified_at DESC;

-- name: GetPublicGames :many
SELECT * FROM game WHERE public = true ORDER BY created_at DESC;

-- name: GetPublicGamesSortedByName :many
SELECT * FROM game WHERE public = true ORDER BY LOWER(name) ASC;

-- name: GetPublicGamesSortedByNameDesc :many
SELECT * FROM game WHERE public = true ORDER BY LOWER(name) DESC;

-- name: GetPublicGamesSortedByCreatedAt :many
SELECT * FROM game WHERE public = true ORDER BY created_at ASC;

-- name: GetPublicGamesSortedByModifiedAt :many
SELECT * FROM game WHERE public = true ORDER BY modified_at DESC;

-- name: GetPublicGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE public = true ORDER BY modified_at ASC;

-- name: SearchPublicGames :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY created_at DESC;

-- name: SearchPublicGamesSortedByName :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY LOWER(name) ASC;

-- name: SearchPublicGamesSortedByNameDesc :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY LOWER(name) DESC;

-- name: SearchPublicGamesSortedByCreatedAt :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY created_at ASC;

-- name: SearchPublicGamesSortedByModifiedAt :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY modified_at DESC;

-- name: SearchPublicGamesSortedByModifiedAtAsc :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY modified_at ASC;

-- name: GetPublicGamesSortedByPlayCount :many
SELECT * FROM game WHERE public = true ORDER BY play_count DESC;

-- name: GetPublicGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE public = true ORDER BY play_count ASC;

-- name: SearchPublicGamesSortedByPlayCount :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY play_count DESC;

-- name: SearchPublicGamesSortedByPlayCountAsc :many
SELECT * FROM game WHERE public = true AND LOWER(name) LIKE LOWER('%' || $1 || '%') ORDER BY play_count ASC;

-- Games visible to user with additional sort options
-- name: GetGamesVisibleToUserSortedByPlayCount :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY play_count DESC;

-- name: GetGamesVisibleToUserSortedByPlayCountAsc :many
SELECT * FROM game WHERE created_by = $1 OR public = true ORDER BY play_count ASC;

-- name: SearchGamesVisibleToUserSortedByPlayCount :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY play_count DESC;

-- name: SearchGamesVisibleToUserSortedByPlayCountAsc :many
SELECT * FROM game WHERE (created_by = $1 OR public = true) AND LOWER(name) LIKE LOWER('%' || $2 || '%') ORDER BY play_count ASC;

-- Creator sorting requires joining with user table
-- name: GetGamesVisibleToUserSortedByCreator :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE g.created_by = $1 OR g.public = true ORDER BY LOWER(COALESCE(u.name, '')) ASC;

-- name: GetGamesVisibleToUserSortedByCreatorDesc :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE g.created_by = $1 OR g.public = true ORDER BY LOWER(COALESCE(u.name, '')) DESC;

-- name: SearchGamesVisibleToUserSortedByCreator :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE (g.created_by = $1 OR g.public = true) AND LOWER(g.name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(COALESCE(u.name, '')) ASC;

-- name: SearchGamesVisibleToUserSortedByCreatorDesc :many
SELECT g.* FROM game g LEFT JOIN app_user u ON g.created_by = u.id WHERE (g.created_by = $1 OR g.public = true) AND LOWER(g.name) LIKE LOWER('%' || $2 || '%') ORDER BY LOWER(COALESCE(u.name, '')) DESC;

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
  first_image = $20,
  originally_created_by = $21
WHERE id = $1
RETURNING *;

-- name: IncrementGamePlayCount :exec
UPDATE game SET play_count = play_count + 1 WHERE id = $1;

-- name: IncrementGameCloneCount :exec
UPDATE game SET clone_count = clone_count + 1 WHERE id = $1;

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
  game_id, user_id, workshop_id, api_key_id,
  ai_platform, ai_model, ai_session,
  image_style, status_fields, theme
) VALUES (
  gen_random_uuid(), $1,
  $2, $3, $4,
<<<<<<< HEAD
  $5, $6, $7,
  $8, $9, $10,
  $11, $12, $13
=======
  $5, $6, $7, $8,
  $9, $10, $11,
  $12, $13
>>>>>>> 876746c (feat: centralized db permissions)
)
RETURNING *;

-- name: GetGameSessionByID :one
SELECT * FROM game_session WHERE id = $1;

-- name: GetGameSessionsByGameID :many
SELECT * FROM game_session WHERE game_id = $1 ORDER BY created_at DESC;

-- name: GetGameSessionsByUserID :many
SELECT 
  gs.*,
  g.name as game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1 
ORDER BY gs.modified_at DESC
LIMIT 20;

-- name: GetGameSessionsByUserIDSortByGame :many
SELECT 
  gs.*,
  g.name as game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1 
ORDER BY LOWER(g.name) ASC, gs.modified_at DESC
LIMIT 20;

-- name: GetGameSessionsByUserIDSortByModel :many
SELECT 
  gs.*,
  g.name as game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1 
ORDER BY gs.ai_model ASC, gs.modified_at DESC
LIMIT 20;

-- name: SearchGameSessionsByUserID :many
SELECT 
  gs.*,
  g.name as game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1 AND LOWER(g.name) LIKE LOWER('%' || $2 || '%')
ORDER BY gs.modified_at DESC
LIMIT 20;

-- name: SearchGameSessionsByUserIDSortByGame :many
SELECT 
  gs.*,
  g.name as game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1 AND LOWER(g.name) LIKE LOWER('%' || $2 || '%')
ORDER BY LOWER(g.name) ASC, gs.modified_at DESC
LIMIT 20;

-- name: SearchGameSessionsByUserIDSortByModel :many
SELECT 
  gs.*,
  g.name as game_name
FROM game_session gs
JOIN game g ON gs.game_id = g.id
WHERE gs.user_id = $1 AND LOWER(g.name) LIKE LOWER('%' || $2 || '%')
ORDER BY gs.ai_model ASC, gs.modified_at DESC
LIMIT 20;

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
  status_fields = $13,
  theme = $14
WHERE id = $1
RETURNING *;

-- name: UpdateGameSessionTheme :exec
UPDATE game_session SET
  modified_at = now(),
  theme = $2
WHERE id = $1;

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

-- name: GetLatestGameSessionMessage :one
SELECT * FROM game_session_message WHERE game_session_id = $1 ORDER BY seq DESC LIMIT 1;

-- name: GetAllGameSessionMessages :many
SELECT * FROM game_session_message WHERE game_session_id = $1 ORDER BY seq ASC;

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

-- name: DeleteGameSessionMessagesBySessionID :exec
DELETE FROM game_session_message WHERE game_session_id = $1;

-- name: DeleteUserGameSessions :exec
DELETE FROM game_session WHERE user_id = $1 AND game_id = $2;
