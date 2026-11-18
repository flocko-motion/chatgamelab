-- admin statistics --------------------------------------------------------
-- Months are calendar months in Europe/Berlin.

-- name: RecordPlayStart :exec
-- One row per started session. The organisation comes from the share link if there is one,
-- else from the player's role (directly or via their workshop).
INSERT INTO play_stat (session_id, game_id, institution_id, workshop_id, via_share)
SELECT sqlc.arg(session_id)::uuid, sqlc.arg(game_id)::uuid,
       COALESCE(sqlc.narg(share_institution_id)::uuid, sw.institution_id, r.institution_id),
       COALESCE(sqlc.narg(share_workshop_id)::uuid, r.workshop_id),
       sqlc.arg(via_share)::boolean
FROM (SELECT 1) AS one
LEFT JOIN workshop sw ON sw.id = sqlc.narg(share_workshop_id)::uuid
LEFT JOIN LATERAL (
  SELECT COALESCE(ur.institution_id, w.institution_id) AS institution_id,
         COALESCE(ur.workshop_id, ur.active_workshop_id) AS workshop_id
  FROM user_role ur
  LEFT JOIN workshop w ON w.id = COALESCE(ur.workshop_id, ur.active_workshop_id)
  WHERE ur.user_id = sqlc.arg(user_id)::uuid
    AND COALESCE(ur.institution_id, w.institution_id) IS NOT NULL
  LIMIT 1
) r ON true
ON CONFLICT (session_id) DO NOTHING;

-- name: RecordPlayTurn :exec
UPDATE play_stat SET turns = turns + 1, last_turn_at = now() WHERE session_id = $1;

-- name: StatsInstitutionCounts :one
SELECT
  count(*)::int AS total,
  count(*) FILTER (WHERE disabled_at IS NOT NULL)::int AS locked
FROM institution WHERE deleted_at IS NULL;

-- name: StatsInstitutionsPerMonth :many
SELECT to_char(date_trunc('month', created_at AT TIME ZONE 'Europe/Berlin'), 'YYYY-MM')::text AS month,
       count(*)::int AS n
FROM institution
WHERE deleted_at IS NULL AND created_at >= sqlc.arg(since)::timestamptz
GROUP BY 1;

-- name: StatsGamesPerInstitutionMonth :many
-- New games per organisation and month; the nil UUID stands for games outside organisations.
-- A game belongs to the organisation of its workshop, else to its creator's organisation.
SELECT
  COALESCE(w.institution_id, (
    SELECT COALESCE(ur.institution_id, uw.institution_id)
    FROM user_role ur LEFT JOIN workshop uw ON uw.id = ur.workshop_id
    WHERE ur.user_id = g.created_by AND COALESCE(ur.institution_id, uw.institution_id) IS NOT NULL
    LIMIT 1), '00000000-0000-0000-0000-000000000000'::uuid)::uuid AS institution_id,
  to_char(date_trunc('month', g.created_at AT TIME ZONE 'Europe/Berlin'), 'YYYY-MM')::text AS month,
  (g.created_at >= sqlc.arg(since7)::timestamptz) AS last7,
  (g.created_at >= sqlc.arg(since30)::timestamptz) AS last30,
  count(*)::int AS n
FROM game g
LEFT JOIN workshop w ON w.id = g.workshop_id
WHERE g.created_at >= sqlc.arg(since)::timestamptz
GROUP BY 1, 2, 3, 4;

-- name: StatsPlaysPerInstitutionMonth :many
-- Started sessions and AI turns per organisation and month (NULL = outside organisations).
SELECT
  institution_id,
  to_char(date_trunc('month', created_at AT TIME ZONE 'Europe/Berlin'), 'YYYY-MM')::text AS month,
  (created_at >= sqlc.arg(since7)::timestamptz) AS last7,
  (created_at >= sqlc.arg(since30)::timestamptz) AS last30,
  count(*)::int AS sessions,
  COALESCE(sum(turns), 0)::int AS turns
FROM play_stat
WHERE created_at >= sqlc.arg(since)::timestamptz
GROUP BY 1, 2, 3, 4;

-- name: StatsPlayStatSince :one
-- The first recorded play (statistics on plays start there).
-- Without any play yet the current time is returned.
SELECT COALESCE(min(created_at), now())::timestamptz AS first_play FROM play_stat;

-- name: StatsInstitutionOverview :many
-- Per organisation: members by role, workshops, games, last activity.
SELECT
  i.id, i.name, i.city, i.created_at, i.disabled_at,
  (SELECT count(DISTINCT ur.user_id) FROM user_role ur WHERE ur.institution_id = i.id AND ur.role = 'head')::int AS heads,
  (SELECT count(DISTINCT ur.user_id) FROM user_role ur WHERE ur.institution_id = i.id AND ur.role = 'staff')::int AS staff,
  (SELECT count(DISTINCT ur.user_id) FROM user_role ur WHERE ur.institution_id = i.id AND ur.role = 'individual')::int AS individuals,
  (SELECT count(DISTINCT ur.user_id) FROM user_role ur JOIN workshop w ON w.id = ur.workshop_id
     WHERE w.institution_id = i.id AND ur.role = 'participant')::int AS participants,
  (SELECT count(*) FROM workshop w WHERE w.institution_id = i.id)::int AS workshops,
  (SELECT count(*) FROM workshop w WHERE w.institution_id = i.id AND w.active AND NOT w.is_paused)::int AS active_workshops,
  (SELECT count(*) FROM game g
     LEFT JOIN workshop gw ON gw.id = g.workshop_id
     WHERE g.deleted_at IS NULL AND (gw.institution_id = i.id OR (g.workshop_id IS NULL AND EXISTS (
       SELECT 1 FROM user_role ur WHERE ur.user_id = g.created_by AND ur.institution_id = i.id))))::int AS games,
  -- 'epoch' = no recorded activity yet
  COALESCE((SELECT max(GREATEST(p.created_at, COALESCE(p.last_turn_at, p.created_at))) FROM play_stat p WHERE p.institution_id = i.id), 'epoch'::timestamptz)::timestamptz AS last_activity
FROM institution i
WHERE i.deleted_at IS NULL
ORDER BY lower(i.name);

-- name: StatsRegistrationFunnel :one
-- Self-registrations since a date: submitted, confirmed, and confirmed with at least one workshop.
SELECT
  count(*)::int AS submitted,
  count(*) FILTER (WHERE a.status = 'confirmed')::int AS confirmed,
  count(*) FILTER (WHERE a.status = 'confirmed' AND EXISTS (
    SELECT 1 FROM workshop w WHERE w.institution_id = a.institution_id))::int AS with_workshop
FROM institution_application a
WHERE a.created_at >= sqlc.arg(since)::timestamptz;

-- name: RecordShareView :exec
-- One anonymous page view of a share link. The organisation comes from the share
-- or from the share's workshop.
INSERT INTO share_view (game_id, share_id, institution_id, workshop_id)
SELECT sqlc.arg(game_id)::uuid, sqlc.narg(share_id)::uuid,
       COALESCE(sqlc.narg(institution_id)::uuid, w.institution_id),
       sqlc.narg(workshop_id)::uuid
FROM (SELECT 1) AS one
LEFT JOIN workshop w ON w.id = sqlc.narg(workshop_id)::uuid;

-- name: StatsShareViewSince :one
-- The first recorded share page view; without any view yet the current time.
SELECT COALESCE(min(created_at), now())::timestamptz AS first_view FROM share_view;

-- name: StatsShareTotals :one
-- Share page views and sessions started via a share link, last 7 and 30 days.
SELECT
  (SELECT count(*) FROM share_view WHERE created_at >= sqlc.arg(since7)::timestamptz)::int AS views7,
  (SELECT count(*) FROM share_view WHERE created_at >= sqlc.arg(since30)::timestamptz)::int AS views30,
  (SELECT count(*) FROM play_stat WHERE via_share AND created_at >= sqlc.arg(since7)::timestamptz)::int AS starts7,
  (SELECT count(*) FROM play_stat WHERE via_share AND created_at >= sqlc.arg(since30)::timestamptz)::int AS starts30;

-- name: StatsShareViewsPerMonth :many
SELECT to_char(date_trunc('month', created_at AT TIME ZONE 'Europe/Berlin'), 'YYYY-MM')::text AS month,
       count(*)::int AS n
FROM share_view
WHERE created_at >= sqlc.arg(since)::timestamptz
GROUP BY 1;

-- name: StatsSharedGames :many
-- Games with share page views in the last 30 days, most viewed first:
-- views and starts via a share link, last 7 and 30 days.
WITH v AS (
  SELECT game_id,
         count(*) FILTER (WHERE created_at >= sqlc.arg(since7)::timestamptz) AS views7,
         count(*) AS views30
  FROM share_view WHERE created_at >= sqlc.arg(since30)::timestamptz
  GROUP BY game_id
), s AS (
  SELECT game_id,
         count(*) FILTER (WHERE created_at >= sqlc.arg(since7)::timestamptz) AS starts7,
         count(*) AS starts30
  FROM play_stat WHERE via_share AND game_id IS NOT NULL AND created_at >= sqlc.arg(since30)::timestamptz
  GROUP BY game_id
)
SELECT v.game_id, COALESCE(g.name, '')::text AS name, (g.deleted_at IS NOT NULL OR g.id IS NULL)::boolean AS deleted,
       v.views7::int AS views7, v.views30::int AS views30,
       COALESCE(s.starts7, 0)::int AS starts7, COALESCE(s.starts30, 0)::int AS starts30
FROM v
LEFT JOIN s ON s.game_id = v.game_id
LEFT JOIN game g ON g.id = v.game_id
ORDER BY v.views30 DESC, name
LIMIT sqlc.arg(max_rows)::int;
