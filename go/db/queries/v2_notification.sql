-- v2: the secure core's queries for notifications (#578): per type, the
-- facts that are news to a user, from where their notifications of it start
-- (notification_start). core/notification.go has checked that the caller
-- holds the type.

-- name: V2NewOrganisationsNotice :one
-- The organisations created since the start, by anyone but the user.
SELECT count(i.id)::int AS count, COALESCE(max(i.created_at), now())::timestamptz AS at
FROM institution i
WHERE i.deleted_at IS NULL
  AND i.created_by IS DISTINCT FROM @user_id::uuid
  AND i.created_at > notification_start(@user_id::uuid, @type::text, NULL);

-- name: V2PagesWaitingNotice :one
-- The verified pages asked for since the start and still waiting: not
-- approved, and not turned down since they were asked for.
SELECT count(v.id)::int AS count, COALESCE(max(v.requested_at), now())::timestamptz AS at
FROM verified_page v
WHERE v.approved_at IS NULL
  AND (v.turned_down_at IS NULL OR v.turned_down_at < v.requested_at)
  AND v.requested_at > notification_start(@user_id::uuid, @type::text, NULL);

-- name: V2PageApprovedNotice :many
-- Organisation institution_id's verified page, at most one, if an admin
-- approved it since the start.
SELECT v.approved_at::timestamptz AS at
FROM verified_page v
WHERE v.institution_id = @institution_id::uuid
  AND v.approved_at > notification_start(@user_id::uuid, @type::text, NULL);

-- name: V2PageTurnedDownNotice :many
-- Organisation institution_id's verified page, at most one, if an admin
-- turned it down or withdrew its approval since the start, and it was
-- neither approved nor asked for again since.
SELECT v.turned_down_at::timestamptz AS at
FROM verified_page v
WHERE v.institution_id = @institution_id::uuid
  AND v.approved_at IS NULL
  AND v.turned_down_at >= v.requested_at
  AND v.turned_down_at > notification_start(@user_id::uuid, @type::text, NULL);

-- name: V2AcknowledgeNotification :exec
-- The user acknowledges a type (and subject) up to a time, at most now; an
-- acknowledgement never moves back.
INSERT INTO notification_ack (user_id, type, subject, acked_at)
VALUES (@user_id::uuid, @type::text, sqlc.narg(subject)::uuid,
        LEAST(COALESCE(sqlc.narg(up_to)::timestamptz, now()), now()))
ON CONFLICT ON CONSTRAINT notification_ack_uniq
DO UPDATE SET acked_at = GREATEST(notification_ack.acked_at, EXCLUDED.acked_at);
