-- v2: the secure core's caller (#344).
--
-- One round trip for everything the core needs to know about who is asking:
-- the facts the rules read, and the account states refused on every route.

-- name: V2ReadCaller :one
-- The role is chosen exactly as v1 does (GetUserDetailsByID): one role per
-- account, a guest's last, newest first.
SELECT
  u.id,
  (u.deleted_at IS NOT NULL)::bool                              AS deleted,
  (COALESCE(u.name, '') = '')::bool                             AS registering,
  (u.guest_token IS NOT NULL)::bool                             AS guest_account,
  r.role,
  r.institution_id,
  r.workshop_id,
  r.active_workshop_id,
  COALESCE(pw.active AND pw.deleted_at IS NULL, false)::bool    AS guest_workshop_active,
  COALESCE(cw.show_public_games, false)::bool                   AS workshop_shows_public,
  COALESCE(cw.show_other_participants_games, false)::bool       AS workshop_shows_others,
  COALESCE(cw.allow_game_sharing, false)::bool                  AS workshop_allows_sharing
FROM app_user u
LEFT JOIN user_role r ON r.id = (
  SELECT ur.id
  FROM user_role ur
  WHERE ur.user_id = u.id
  ORDER BY CASE WHEN ur.role = 'guest' THEN 1 ELSE 0 END, ur.created_at DESC
  LIMIT 1
)
LEFT JOIN workshop pw ON pw.id = r.workshop_id AND r.role = 'guest'
-- the workshop the caller is in: a guest's own, or the one entered in workshop mode
LEFT JOIN workshop cw ON cw.id = CASE WHEN r.role = 'guest' THEN r.workshop_id ELSE r.active_workshop_id END
WHERE u.id = @id;
