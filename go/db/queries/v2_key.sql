-- v2: the secure core's queries for API keys and key shares (#344, #451).
-- core/key*.go has checked that the caller may; these read and write what
-- they are told. A share is permission to use a key: the owner's own (the
-- self-share), given to an organisation, or drawn for a sponsored game or a
-- share link (game_id set, source_share_id the share it draws from; NULL:
-- the key itself).

-- name: V2Keys :many
-- The keys the caller owns, for a list or for one key (id set). The secret
-- never leaves the database: only its last four characters do.
SELECT k.id, k.name, k.platform, right(k.key, 4)::text AS key_tail, k.is_default,
  k.last_usage_success, k.last_error_code, k.created_at, k.modified_at,
  k.user_id AS owner_id, COALESCE(u.name, '')::text AS owner_name
FROM api_key k
JOIN app_user u ON u.id = k.user_id
WHERE @own::bool AND k.user_id = sqlc.narg(caller)::uuid
  AND (sqlc.narg(id)::uuid IS NULL OR k.id = sqlc.narg(id)::uuid)
ORDER BY k.created_at, k.id;

-- name: V2KeyShares :many
-- The shares the caller may see, for a list or for one share (id set):
-- PaidByOwnKey, every share of their keys; SharedWithOwnOrganisation, those
-- given to their organisation; PaidByOwnOrganisation, those drawn from such a
-- share. Wishes narrow it: one key, one organisation's.
SELECT s.id, s.created_at, s.created_by, s.api_key_id, s.source_share_id,
  k.name AS key_name, k.platform, right(k.key, 4)::text AS key_tail, k.is_default,
  k.last_usage_success, k.last_error_code, k.created_at AS key_created_at, k.modified_at AS key_modified_at,
  k.user_id AS owner_id, COALESCE(u.name, '')::text AS owner_name,
  s.institution_id, o.name AS organisation_name,
  s.game_id, g.name AS game_name,
  gs.id AS link_id, gs.remaining AS link_remaining, gs.workshop_id AS link_workshop_id,
  (s.user_id = k.user_id AND s.institution_id IS NULL AND s.game_id IS NULL)::bool AS is_self,
  -- the actions
  COALESCE(
     (@share_own::bool          AND k.user_id = sqlc.narg(caller)::uuid AND s.game_id IS NULL)
  OR (@share_organisation::bool AND s.institution_id = sqlc.narg(own_organisation)::uuid)
     -- the key of the workshop the caller is in
  OR (@share_workshop::bool AND s.id = (SELECT cw.default_api_key_share_id FROM workshop cw
                                        WHERE cw.id = sqlc.narg(current_workshop)::uuid)), false)::bool AS may_share,
  COALESCE(
     -- the owner takes any share of their key back, their own use aside
     (@revoke_own::bool AND k.user_id = sqlc.narg(caller)::uuid
                        AND NOT (s.user_id = k.user_id AND s.institution_id IS NULL AND s.game_id IS NULL))
     -- a head removes a key from their organisation
  OR (@revoke_organisation::bool AND s.institution_id = sqlc.narg(own_organisation)::uuid)
     -- head and staff stop what draws from their organisation's keys
  OR (@revoke_drawn::bool AND src.institution_id = sqlc.narg(own_organisation)::uuid), false)::bool AS may_revoke
FROM api_key_share s
JOIN api_key k           ON k.id = s.api_key_id
JOIN app_user u          ON u.id = k.user_id
LEFT JOIN api_key_share src ON src.id = s.source_share_id
LEFT JOIN institution o  ON o.id = s.institution_id
LEFT JOIN game g         ON g.id = s.game_id
LEFT JOIN game_share gs  ON gs.api_key_share_id = s.id
WHERE (sqlc.narg(id)::uuid IS NULL OR s.id = sqlc.narg(id)::uuid)
  -- wishes
  AND (sqlc.narg(key_id)::uuid IS NULL OR s.api_key_id = sqlc.narg(key_id)::uuid)
  AND (sqlc.narg(organisation)::uuid IS NULL OR s.institution_id = sqlc.narg(organisation)::uuid)
  -- gate
  AND (  (@card_own::bool          AND k.user_id = sqlc.narg(caller)::uuid)
      OR (@card_organisation::bool AND s.institution_id = sqlc.narg(own_organisation)::uuid)
      OR (@card_drawn::bool        AND src.institution_id = sqlc.narg(own_organisation)::uuid) )
ORDER BY s.created_at, s.id;

-- name: V2CreateKey :exec
INSERT INTO api_key (id, created_by, modified_by, user_id, name, platform, key, is_default)
VALUES (@id, @owner::uuid, @owner::uuid, @owner::uuid, @name, @platform, @key, @is_default);

-- name: V2CreateKeyShare :one
-- A share of key @api_key_id: the owner's own (user_id), given to an
-- organisation (institution_id), or drawn for a game (game_id, source).
INSERT INTO api_key_share (id, created_by, modified_by, api_key_id, user_id, institution_id, game_id, source_share_id)
VALUES (gen_random_uuid(), @created_by::uuid, @created_by::uuid, @api_key_id,
        sqlc.narg(user_id)::uuid, sqlc.narg(institution_id)::uuid, sqlc.narg(game_id)::uuid, sqlc.narg(source_share_id)::uuid)
RETURNING id;

-- name: V2CountOwnKeys :one
SELECT COUNT(*)::int FROM api_key WHERE user_id = @owner;

-- name: V2RenameKey :exec
UPDATE api_key SET name = @name::text, modified_by = @modified_by::uuid, modified_at = now() WHERE id = @id;

-- name: V2ClearDefaultKey :exec
-- None of the owner's keys is their default (before V2SetDefaultKey: one
-- default per owner is a unique index).
UPDATE api_key SET is_default = false, modified_at = now()
WHERE user_id = @owner::uuid AND is_default;

-- name: V2SetDefaultKey :exec
UPDATE api_key SET is_default = true, modified_at = now() WHERE id = @id::uuid;

-- name: V2PromoteOldestKey :exec
-- The owner's oldest key becomes their default, when none is.
UPDATE api_key SET is_default = true, modified_at = now()
WHERE api_key.id = (SELECT o.id FROM api_key o WHERE o.user_id = @owner::uuid ORDER BY o.created_at, o.id LIMIT 1)
  AND NOT EXISTS (SELECT 1 FROM api_key d WHERE d.user_id = @owner::uuid AND d.is_default);

-- name: V2SharesDrawnFrom :many
-- Share @id and the shares that draw from it, and the share links they pay
-- for: what taking it back stops.
SELECT s.id, gs.id AS link_id
FROM api_key_share s
LEFT JOIN game_share gs ON gs.api_key_share_id = s.id
WHERE s.id = @id::uuid OR s.source_share_id = @id::uuid;

-- name: V2SharesOfKey :many
-- Every share of key @key_id and the share links they pay for: what deleting
-- the key ends.
SELECT s.id, gs.id AS link_id
FROM api_key_share s
LEFT JOIN game_share gs ON gs.api_key_share_id = s.id
WHERE s.api_key_id = @key_id::uuid;

-- name: V2DeleteKeyShares :exec
-- Deletes shares, the ones that draw from others first. The pointers of
-- workshops, organisations and games follow them (ON DELETE SET NULL); a
-- user's default and a session's record are cleared here.
WITH cleared_user AS (
  UPDATE app_user SET default_api_key_share_id = NULL WHERE default_api_key_share_id = ANY(@ids::uuid[])
), cleared_session AS (
  UPDATE game_session SET api_key_share_id = NULL WHERE api_key_share_id = ANY(@ids::uuid[])
)
DELETE FROM api_key_share WHERE id = ANY(@ids::uuid[]);

-- name: V2DeleteKey :exec
-- Deletes a key whose shares are gone, and the site's free use of it.
WITH cleared_site AS (
  UPDATE system_settings SET free_use_api_key_id = NULL WHERE free_use_api_key_id = @id::uuid
)
DELETE FROM api_key WHERE id = @id::uuid;

-- name: V2SetWorkshopKey :exec
-- The share a workshop plays with; NULL: none, it does not play.
UPDATE workshop SET default_api_key_share_id = sqlc.narg(share_id)::uuid,
  modified_by = @modified_by::uuid, modified_at = now()
WHERE id = @id::uuid;

-- name: V2SetOrganisationFreeUseKey :exec
-- The share an organisation's members play with for free; NULL: none.
UPDATE institution SET free_use_api_key_share_id = sqlc.narg(share_id)::uuid,
  modified_by = @modified_by::uuid, modified_at = now()
WHERE id = @id::uuid;

-- name: V2KeyShareFacts :one
-- What a share is given to and drawn from, for the core's own decisions.
SELECT id, api_key_id, institution_id, game_id, source_share_id
FROM api_key_share WHERE id = @id;

-- name: V2PayerShare :one
-- The key behind a share, as a session or a share link pays with it.
SELECT s.id, k.id AS key_id, k.name, k.platform, k.last_usage_success
FROM api_key_share s JOIN api_key k ON k.id = s.api_key_id
WHERE s.id = @id;

-- name: V2PayerKey :one
-- A key as a session pays with it without a share: the site's free use.
SELECT id, name, platform FROM api_key WHERE id = @id;

-- name: V2OwnDefaultKeyShare :many
-- The caller's default key, by their own use of it.
SELECT s.id FROM api_key k
JOIN api_key_share s ON s.api_key_id = k.id AND s.user_id = k.user_id AND s.institution_id IS NULL AND s.game_id IS NULL
WHERE k.user_id = @owner AND k.is_default
LIMIT 1;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2DeleteApiKeySharesByGameID :exec
DELETE FROM api_key_share WHERE game_id = $1;

-- name: V2DeleteApiKeySharesByOwnerForInstitution :exec
-- Delete all API key shares owned by a user that target a specific institution
DELETE FROM api_key_share s
WHERE s.api_key_id IN (SELECT k.id FROM api_key k WHERE k.user_id = $1)
  AND s.institution_id = $2;

-- name: V2DeleteApiKeySharesByOwnerForInstitutionWorkshops :exec
-- Delete all API key shares owned by a user that target any workshop in a specific institution
DELETE FROM api_key_share s
WHERE s.api_key_id IN (SELECT k.id FROM api_key k WHERE k.user_id = $1)
  AND s.workshop_id IN (SELECT w.id FROM workshop w WHERE w.institution_id = $2);

-- name: V2GetApiKeyByID :one
SELECT * FROM api_key WHERE id = $1;

-- name: V2UpdateApiKeyLastUsageSuccess :exec
UPDATE api_key SET last_usage_success = $2, last_error_code = $3, modified_at = now()
WHERE id = $1;

-- name: V2OwnKeyPlatforms :many
-- The platforms of a user's own keys (the dev preseed).
SELECT id, platform FROM api_key WHERE user_id = @user_id;
