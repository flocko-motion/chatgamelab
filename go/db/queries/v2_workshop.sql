-- v2: the secure core's queries for workshops (#344). core/workshop*.go has
-- checked that the caller may; these read and write what they are told.

-- name: V2Workshops :many
-- The workshops the caller may see, for a list or for one workshop (id set):
-- the List grants on WorkshopCard for a list, the Read grants for one.
-- Wishes narrow it: one organisation, a name containing @search, active
-- ones only.
SELECT w.id, w.name, w.institution_id, o.name AS organisation_name,
  w.created_at, w.modified_at,
  -- card
  w.active, w.is_paused, w.public, w.public_slug, w.ai_quality_tier,
  w.show_public_games, w.show_other_participants_games, w.allow_game_sharing,
  -- settings
  w.prompt_constraints, w.default_api_key_share_id, w.public_description, w.public_links,
  -- the sections: NULL (no caller, no organisation, no workshop) is false
  COALESCE(
     (@settings_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @settings_everywhere::bool, false)::bool                            AS may_read_settings,
  COALESCE(
     (@participants_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR (@participants_current::bool      AND w.id = sqlc.narg(current_workshop)::uuid)
  OR  @participants_everywhere::bool, false)::bool                        AS may_list_participants,
  -- the actions
  COALESCE(
     (@edit_card_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_card_everywhere::bool, false)::bool                           AS may_edit_card,
  COALESCE(
     (@edit_settings_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @edit_settings_everywhere::bool, false)::bool                       AS may_edit_settings,
  COALESCE(
     (@delete_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @delete_everywhere::bool, false)::bool                              AS may_delete,
  COALESCE(
     (@manage_participants_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
  OR  @manage_participants_everywhere::bool, false)::bool                 AS may_manage_participants
FROM workshop w
JOIN institution o ON o.id = w.institution_id
WHERE w.deleted_at IS NULL
  AND (sqlc.narg(id)::uuid IS NULL OR w.id = sqlc.narg(id)::uuid)
  -- wishes
  AND (sqlc.narg(organisation)::uuid IS NULL OR w.institution_id = sqlc.narg(organisation)::uuid)
  AND (sqlc.narg(search)::text IS NULL OR position(lower(sqlc.narg(search)::text) IN lower(w.name)) > 0)
  AND (NOT @active_only::bool OR w.active)
  -- gate
  AND (  (@card_organisation::bool AND w.institution_id = sqlc.narg(own_organisation)::uuid)
      OR (@card_current::bool      AND w.id = sqlc.narg(current_workshop)::uuid)
      OR (@card_public::bool       AND w.public)
      OR  @card_everywhere::bool )
ORDER BY w.created_at DESC, w.id;

-- name: V2CreateWorkshop :exec
-- A new workshop shows its participants nothing beyond their own games and
-- lets them share nothing, until its staff switch that on (v1).
INSERT INTO workshop (
  id, created_by, modified_by, name, institution_id, active, public, public_slug,
  show_public_games, show_other_participants_games, is_paused, allow_game_sharing
) VALUES (
  @id, @creator::uuid, @creator::uuid, @name, @institution_id, @active, @public, @public_slug::text,
  false, false, false, false
);

-- name: V2UpdateWorkshop :exec
-- Writes a workshop's card and settings as the caller changed them;
-- core/workshop_update.go has checked that they may. Its key is apikeys.go's
-- until key shares are ported (#451).
UPDATE workshop SET
  -- card
  name                          = @name::text,
  active                        = @active::bool,
  is_paused                     = @is_paused::bool,
  public                        = @public::bool,
  public_slug                   = @public_slug::text,
  ai_quality_tier               = sqlc.narg(ai_quality_tier)::text,
  show_public_games             = @show_public_games::bool,
  show_other_participants_games = @show_other_participants_games::bool,
  allow_game_sharing            = @allow_game_sharing::bool,
  -- settings
  prompt_constraints            = sqlc.narg(prompt_constraints)::text,
  public_description            = sqlc.narg(public_description)::text,
  public_links                  = sqlc.narg(public_links)::jsonb,
  modified_by                   = @modified_by::uuid,
  modified_at                   = now()
WHERE id = @id::uuid AND deleted_at IS NULL;

-- name: V2WorkshopNameTaken :one
-- Whether another workshop of the organisation has this name (the column is
-- unique per organisation).
SELECT EXISTS (
  SELECT 1 FROM workshop
  WHERE institution_id = @institution_id AND name = @name::text AND id <> @id::uuid
)::bool AS taken;

-- name: V2PublicWorkshop :many
-- The workshop behind a public page's slug, while its page is switched on
-- and its organisation is not suspended (by an admin, or for want of a
-- consent to @declaration_*).
SELECT w.id, w.name, w.public_description, w.public_links, w.default_api_key_share_id,
  w.institution_id, w.created_by, ks.created_by AS key_share_created_by
FROM workshop w
JOIN institution i ON i.id = w.institution_id
LEFT JOIN api_key_share ks ON ks.id = w.default_api_key_share_id
WHERE w.public_slug = @slug AND w.deleted_at IS NULL AND w.public
  AND i.suspended_at IS NULL
  AND NOT institution_consent_missing(i.id, @declaration_version::text, @declaration_hash::text);

-- name: V2PublicSlugExists :one
-- Deleted workshops keep their slug, so an old link never points at a new workshop.
SELECT EXISTS(SELECT 1 FROM workshop WHERE public_slug = @slug)::bool;

-- name: V2PublicWorkshopGames :many
-- The games on a workshop's public page: its public ones; playable when
-- their scenario and opening are written.
SELECT g.id, g.name, g.description,
  (g.system_message_scenario <> '' AND g.system_message_game_start <> '')::bool AS playable
FROM game g
WHERE g.workshop_id = @workshop_id AND g.public AND g.deleted_at IS NULL
ORDER BY LOWER(g.name), g.id;

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2GetWorkshopByID :one
SELECT * FROM workshop WHERE id = $1 AND deleted_at IS NULL;

-- Deleting a workshop (core/delete.go).

-- name: V2WorkshopGuests :many
SELECT user_id FROM user_role WHERE workshop_id = @workshop_id AND role = 'guest';

-- name: V2LeaveDeletedWorkshop :exec
-- Whoever works in the workshop (head, staff, an individual taking part)
-- leaves it.
UPDATE user_role SET active_workshop_id = NULL, modified_at = now() WHERE active_workshop_id = @workshop_id;

-- name: V2UnlinkWorkshopGames :exec
-- Its games stay with their creators.
UPDATE game SET workshop_id = NULL WHERE workshop_id = @workshop_id;

-- name: V2SoftDeleteWorkshop :exec
UPDATE workshop SET deleted_at = now() WHERE id = @id;

-- Carried over from v1 for the core's system jobs (#344).

-- name: V2SeedWorkshop :one
INSERT INTO workshop (
  id, created_by,
  created_at, modified_by, modified_at,
  name, institution_id, active, public, default_api_key_share_id,
  ai_quality_tier, prompt_constraints, show_public_games,
  show_other_participants_games, is_paused,
  allow_game_sharing, public_slug
) VALUES (
  $1, $2,
  $3, $4, $5,
  $6, $7, $8, $9, $10,
  $11, $12, $13,
  $14, $15,
  $16, $17
)
RETURNING *;
