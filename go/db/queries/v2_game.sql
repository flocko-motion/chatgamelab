-- v2: the secure core's game queries (#344).
--
-- Every relation fragment appears once per section; the grant table only
-- switches it on, through one boolean per (section, relation). Which grant
-- opens which fragment is in core/gates.go. The caller's facts arrive as
-- parameters: no join to the user, no lookup per row.

-- name: V2GameCards :many
-- The game cards the caller may see, for a list or for one game (id set).
-- Three blocks, each only narrowing the one before:
--   gate     the List grants on GameCard for a list, the Read grants for one
--            game (core/gates.go); the caller's context (in a workshop, what it
--            shows) is resolved before the query
--   wishes   what the caller asks for: filters (combined), search (core/game.go);
--            the genres asked for are one filter, met by any of them
--   paging   sort order (the id breaks ties, so pages never overlap), page
-- total counts every match across pages, after gate and wishes.
SELECT g.*,
  -- the sections: NULL (no caller, no workshop, no organisation) is false
  COALESCE(
     (@content_public::bool        AND g.public)
  OR (@content_own::bool           AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@content_workshop::bool      AND g.workshop_id    = sqlc.narg(current_workshop)::uuid)
  OR (@content_organisation::bool  AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR (@content_via_share::bool     AND g.id = (
        SELECT gs.game_id FROM game_share gs WHERE gs.token = sqlc.narg(share_token)::text AND gs.enabled))
  OR  @content_everywhere::bool, false)::bool                          AS may_read_content,
  COALESCE(
     (@settings_public::bool       AND g.public)
  OR (@settings_own::bool          AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@settings_workshop::bool     AND g.workshop_id    = sqlc.narg(current_workshop)::uuid)
  OR (@settings_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @settings_everywhere::bool, false)::bool                         AS may_read_settings,
  -- the actions
  COALESCE(
     (@edit_card_own::bool              AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@edit_card_organisation::bool     AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_card_everywhere::bool, false)::bool                        AS may_edit_card,
  COALESCE(
     (@edit_content_own::bool           AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@edit_content_organisation::bool  AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_content_everywhere::bool, false)::bool                     AS may_edit_content,
  COALESCE(
     (@edit_settings_own::bool          AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@edit_settings_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_settings_everywhere::bool, false)::bool                    AS may_edit_settings,
  COALESCE(
     (@delete_own::bool          AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@delete_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @delete_everywhere::bool, false)::bool                         AS may_delete,
  COALESCE(
     (@copy_public::bool  AND g.public)
  OR (@copy_own::bool     AND g.created_by = sqlc.narg(caller)::uuid), false)::bool  AS may_copy,
  COALESCE(
     (@share_public::bool AND g.public)
  OR (@share_own::bool    AND g.created_by = sqlc.narg(caller)::uuid), false)::bool  AS may_share,
  COALESCE(
     (@publish_own::bool          AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@publish_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @publish_everywhere::bool, false)::bool                         AS may_publish,
  COALESCE(
     (@play_public::bool       AND g.public)
  OR (@play_own::bool          AND g.created_by     = sqlc.narg(caller)::uuid)
  OR (@play_workshop::bool     AND g.workshop_id    = sqlc.narg(current_workshop)::uuid)
  OR (@play_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR (@play_via_share::bool    AND g.id = (
        SELECT gs.game_id FROM game_share gs WHERE gs.token = sqlc.narg(share_token)::text AND gs.enabled))
  OR  @play_everywhere::bool, false)::bool                             AS may_play,
  -- the caller's own mark, no right: false for a guest
  EXISTS (SELECT 1 FROM user_favourite_game f
          WHERE f.game_id = g.id AND f.user_id = sqlc.narg(caller)::uuid)  AS is_favorite,
  cu.name                                                                  AS creator_name,
  gc.created_at                                                            AS cover_at,
  count(*) OVER ()                                                         AS total
FROM game g
LEFT JOIN app_user cu ON cu.id = g.created_by  -- the creator's public profile name
LEFT JOIN workshop w  ON w.id  = g.workshop_id -- its organisation, for "of own organisation"
LEFT JOIN game_cover gc ON gc.game_id = g.id   -- when its cover was made, not the picture
WHERE g.deleted_at IS NULL
  AND (sqlc.narg(id)::uuid IS NULL OR g.id = sqlc.narg(id)::uuid)
  -- gate
  AND (  (@card_public::bool           AND g.public)
      OR (@card_own::bool              AND g.created_by     = sqlc.narg(caller)::uuid)
      OR (@card_public_elsewhere::bool AND g.public
                                       AND g.workshop_id IS DISTINCT FROM sqlc.narg(current_workshop)::uuid)
      OR (@card_own_of_workshop::bool  AND g.workshop_id    = sqlc.narg(current_workshop)::uuid
                                       AND g.created_by     = sqlc.narg(caller)::uuid)
      OR (@card_workshop::bool         AND g.workshop_id    = sqlc.narg(current_workshop)::uuid)
      OR (@card_organisation::bool     AND w.institution_id = sqlc.narg(own_organisation)::uuid)
      OR (@card_via_share::bool        AND g.id = (
            SELECT gs.game_id FROM game_share gs WHERE gs.token = sqlc.narg(share_token)::text AND gs.enabled))
      OR  @card_everywhere::bool )
  -- wishes
  AND (NOT @only_own::bool       OR g.created_by = sqlc.narg(caller)::uuid)
  AND (NOT @only_others::bool    OR g.created_by IS DISTINCT FROM sqlc.narg(caller)::uuid)
  AND (NOT @only_public::bool    OR g.public)
  AND (NOT @only_favorites::bool OR EXISTS (
        SELECT 1 FROM user_favourite_game f
        WHERE f.game_id = g.id AND f.user_id = sqlc.narg(caller)::uuid))
  AND (NOT @only_sponsored::bool OR g.public_sponsored_api_key_share_id IS NOT NULL)
  AND (COALESCE(cardinality(@genres::text[]), 0) = 0 OR g.genre = ANY(@genres::text[]))
  AND (@search::text = ''        OR lower(g.name) LIKE '%' || lower(@search::text) || '%')
-- paging
ORDER BY
  CASE WHEN @sort_by::text = 'name'       AND @sort_desc::bool     THEN lower(g.name) END DESC,
  CASE WHEN @sort_by::text = 'name'       AND NOT @sort_desc::bool THEN lower(g.name) END ASC,
  CASE WHEN @sort_by::text = 'modifiedAt' AND @sort_desc::bool     THEN g.modified_at END DESC,
  CASE WHEN @sort_by::text = 'modifiedAt' AND NOT @sort_desc::bool THEN g.modified_at END ASC,
  CASE WHEN @sort_by::text = 'playCount'  AND @sort_desc::bool     THEN g.play_count END DESC,
  CASE WHEN @sort_by::text = 'playCount'  AND NOT @sort_desc::bool THEN g.play_count END ASC,
  CASE WHEN @sort_by::text = 'visibility' AND @sort_desc::bool     THEN g.public END DESC,
  CASE WHEN @sort_by::text = 'visibility' AND NOT @sort_desc::bool THEN g.public END ASC,
  CASE WHEN @sort_by::text = 'creator'    AND @sort_desc::bool     THEN lower(cu.name) END DESC,
  CASE WHEN @sort_by::text = 'creator'    AND NOT @sort_desc::bool THEN lower(cu.name) END ASC,
  CASE WHEN @sort_by::text = 'createdAt'  AND NOT @sort_desc::bool THEN g.created_at END ASC,
  g.created_at DESC,
  g.id
LIMIT sqlc.narg(page_size)::int OFFSET @page_offset::int;

-- name: V2UpdateGame :exec
-- Writes a game's card and content as the caller changed them; core/game.go
-- has checked that they may. A private game has no sponsor.
UPDATE game SET
  modified_by               = @modified_by::uuid,
  modified_at               = now(),
  -- card
  name                      = @name::text,
  description               = @description::text,
  icon                      = @icon::bytea,
  genre                     = @genre::text,
  theme                     = sqlc.narg(theme)::text,
  public                    = @public::bool,
  public_sponsored_api_key_share_id =
    CASE WHEN @public::bool THEN public_sponsored_api_key_share_id END,
  -- content
  system_message_scenario   = @scenario::text,
  system_message_game_start = @game_start::text,
  image_style               = @image_style::text,
  voice_style               = @voice_style::text,
  css                       = @css::text,
  status_fields             = @status_fields::text,
  first_message             = sqlc.narg(first_message)::text,
  first_status              = sqlc.narg(first_status)::text,
  first_image               = @first_image::bytea
WHERE id = @id::uuid AND deleted_at IS NULL;

-- name: V2CreateGame :exec
-- Creates a game from what the caller filled in; core/game_create.go has
-- checked that they may. It belongs to the workshop they are in, if any; a
-- copy names whoever first created the game it copies.
INSERT INTO game (
  id, created_by, created_at, modified_by, modified_at, workshop_id,
  name, description, icon, genre, theme, public,
  system_message_scenario, system_message_game_start, image_style, voice_style, css,
  status_fields, first_message, first_status, first_image, originally_created_by
) VALUES (
  @id::uuid, @created_by::uuid, now(), @created_by::uuid, now(), sqlc.narg(workshop_id)::uuid,
  @name::text, @description::text, @icon::bytea, @genre::text, sqlc.narg(theme)::text, @public::bool,
  @scenario::text, @game_start::text, @image_style::text, @voice_style::text, @css::text,
  @status_fields::text, sqlc.narg(first_message)::text, sqlc.narg(first_status)::text, @first_image::bytea,
  sqlc.narg(originally_created_by)::uuid
);

-- name: V2MarkFavorite :exec
-- Marks a game as one of the caller's favorites; core/game_favorite.go has
-- checked that they may read it. Marking it twice changes nothing.
INSERT INTO user_favourite_game (id, created_by, created_at, modified_by, modified_at, user_id, game_id)
VALUES (gen_random_uuid(), @caller::uuid, now(), @caller::uuid, now(), @caller::uuid, @game_id::uuid)
ON CONFLICT (user_id, game_id) DO NOTHING;

-- name: V2UnmarkFavorite :exec
-- Removes the caller's own mark, whether or not there was one.
DELETE FROM user_favourite_game WHERE user_id = @caller::uuid AND game_id = @game_id::uuid;

-- name: V2GameCover :one
-- A game's cover picture; core/game_cover.go has checked that the caller may
-- read the game's card.
SELECT image, created_at FROM game_cover WHERE game_id = @game_id::uuid;


-- name: V2SetGameCover :exec
-- Stores a game's cover, replacing the one it had; core/game_cover.go has
-- checked that the caller may edit the game's card.
INSERT INTO game_cover (game_id, image, created_by, created_at)
VALUES (@game_id::uuid, @image::bytea, @created_by::uuid, now())
ON CONFLICT (game_id) DO UPDATE SET
  image = EXCLUDED.image, created_by = EXCLUDED.created_by, created_at = now();

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2DeleteFavoritesByGameID :exec
DELETE FROM user_favourite_game WHERE game_id = $1;

-- name: V2DeleteGameTagsByGameID :exec
DELETE FROM game_tag WHERE game_id = $1;

-- name: V2IncrementGameCloneCount :exec
UPDATE game SET clone_count = clone_count + 1 WHERE id = $1;

-- name: V2SoftDeleteGame :exec
UPDATE game SET deleted_at = now(), modified_at = now() WHERE id = $1;

-- name: V2HardDeleteGame :exec
-- A game gone for good, with its deleted creator's account.
DELETE FROM game WHERE id = @id;

-- Carried over from v1 for the core's system jobs (#344).

-- name: V2SeedGameHistory :exec
-- Development seeding only: gives a demo game a past, so lists have
-- something to sort by.
UPDATE game SET created_at = $2, modified_at = $3, play_count = $4 WHERE id = $1;

-- name: V2GameNamesCreatedBy :many
-- A user's games by name (the dev preseed matches demo covers to them).
SELECT id, name FROM game WHERE created_by = @user_id AND deleted_at IS NULL;

-- name: V2DeleteGameCover :exec
-- A game's cover goes with the game, when it goes for good.
DELETE FROM game_cover WHERE game_id = @game_id;

-- name: V2ClearCoverCreator :exec
-- Covers someone drew stay with their games when the account goes.
UPDATE game_cover SET created_by = NULL WHERE created_by = @user_id;
