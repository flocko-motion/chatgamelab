-- v2: the secure core's queries for share links and sponsored games (#344,
-- #451). A share link is a voucher: plays of one game for whoever has its
-- token, paid by the share it was drawn for (game_share.api_key_share_id),
-- which draws from a share of the key (source_share_id). core/share*.go has
-- checked that the caller may.

-- name: V2ShareLinks :many
-- The share links the caller may see, for a list or for one link (id set):
-- those they made, of their games, paid by their keys, paid by their
-- organisation's keys, and their current workshop's. Wishes narrow it: one
-- game, one key. The paying organisation is the source share's, or, for a
-- link v1 made without one, the link's own.
SELECT gs.id, gs.created_at, gs.created_by, gs.token, gs.remaining, gs.ai_quality_tier, gs.public_page,
  gs.game_id, g.name AS game_name,
  gs.workshop_id, w.name AS workshop_name,
  gs.api_key_share_id, c.source_share_id,
  k.id AS key_id, k.name AS key_name, k.platform, k.user_id AS owner_id,
  COALESCE(src.institution_id, gs.institution_id) AS paying_organisation,
  COALESCE(
     (@manage_own::bool          AND gs.created_by = sqlc.narg(caller)::uuid)
  OR (@manage_game::bool         AND g.created_by = sqlc.narg(caller)::uuid)
  OR (@manage_key::bool          AND k.user_id = sqlc.narg(caller)::uuid)
  OR (@manage_organisation::bool AND COALESCE(src.institution_id, gs.institution_id) = sqlc.narg(own_organisation)::uuid),
     false)::bool AS may_manage
FROM game_share gs
JOIN game g             ON g.id = gs.game_id
JOIN api_key_share c    ON c.id = gs.api_key_share_id
JOIN api_key k          ON k.id = c.api_key_id
LEFT JOIN api_key_share src ON src.id = c.source_share_id
LEFT JOIN workshop w    ON w.id = gs.workshop_id
WHERE (sqlc.narg(id)::uuid IS NULL OR gs.id = sqlc.narg(id)::uuid)
  -- wishes
  AND (sqlc.narg(game)::uuid IS NULL OR gs.game_id = sqlc.narg(game)::uuid)
  AND (sqlc.narg(key_id)::uuid IS NULL OR k.id = sqlc.narg(key_id)::uuid)
  -- gate
  AND (  (@card_own::bool          AND gs.created_by = sqlc.narg(caller)::uuid)
      OR (@card_game::bool         AND g.created_by = sqlc.narg(caller)::uuid)
      OR (@card_key::bool          AND k.user_id = sqlc.narg(caller)::uuid)
      OR (@card_organisation::bool AND COALESCE(src.institution_id, gs.institution_id) = sqlc.narg(own_organisation)::uuid)
      -- those in a workshop share its links; the public page's are the page's
      OR (@card_workshop::bool     AND gs.workshop_id = sqlc.narg(current_workshop)::uuid AND NOT gs.public_page) )
ORDER BY gs.created_at DESC, gs.id;

-- name: V2CreateShareLink :one
INSERT INTO game_share (game_id, token, api_key_share_id, institution_id, workshop_id, remaining, ai_quality_tier, created_by)
VALUES (@game_id, @token, @api_key_share_id, sqlc.narg(institution_id)::uuid, sqlc.narg(workshop_id)::uuid,
        sqlc.narg(remaining)::int, sqlc.narg(ai_quality_tier)::text, @created_by::uuid)
RETURNING id;

-- name: V2UpdateShareLink :exec
UPDATE game_share SET remaining = sqlc.narg(remaining)::int, ai_quality_tier = sqlc.narg(ai_quality_tier)::text
WHERE id = @id::uuid;

-- name: V2WorkshopShareLinkOf :many
-- The link a workshop already has for a game, if any: one per game and
-- workshop besides the public page's.
SELECT id FROM game_share
WHERE game_id = @game_id AND workshop_id = @workshop_id AND NOT public_page
ORDER BY created_at
LIMIT 1;

-- name: V2SetGameSponsor :exec
-- The share a game's public plays draw from; NULL: none.
UPDATE game SET public_sponsored_api_key_share_id = sqlc.narg(share_id)::uuid WHERE id = @id::uuid;

-- name: V2GameSponsor :one
-- The share a game's public plays draw from, if any.
SELECT public_sponsored_api_key_share_id FROM game WHERE id = @id::uuid;

-- name: V2PublicPageLinks :many
-- The share links of a workshop's public page, or of one game's on every
-- page; one per game and page.
SELECT gs.id, gs.game_id, gs.token, gs.remaining, gs.api_key_share_id
FROM game_share gs
WHERE gs.public_page
  AND (sqlc.narg(workshop_id)::uuid IS NULL OR gs.workshop_id = sqlc.narg(workshop_id)::uuid)
  AND (sqlc.narg(game_id)::uuid IS NULL OR gs.game_id = sqlc.narg(game_id)::uuid);

-- name: V2CreatePublicPageLink :one
-- A public page's link for one game, paid by @api_key_share_id (drawn from
-- the workshop's key); the unique index game_share_public_page_uniq keeps
-- it one per game and page.
INSERT INTO game_share (game_id, token, api_key_share_id, institution_id, workshop_id, remaining, created_by, public_page)
VALUES (@game_id, @token, @api_key_share_id, @institution_id, @workshop_id, @remaining, @created_by::uuid, true)
RETURNING id;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2DeleteGameShare :exec
DELETE FROM game_share WHERE id = $1;

-- name: V2DeleteGameSharesByGameID :exec
DELETE FROM game_share WHERE game_id = $1;

-- name: V2DeleteLinkGuests :exec
-- Deletes the guests who came through share link @link_id (their sessions
-- go with them), so the link can end.
DELETE FROM app_user WHERE private_share_id = @link_id;

-- name: V2DeleteGameLinkGuests :exec
-- Deletes the guests who came through any share link of game @game_id.
DELETE FROM app_user WHERE private_share_id IN (SELECT id FROM game_share WHERE game_id = @game_id);
