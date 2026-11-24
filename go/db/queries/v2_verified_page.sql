-- v2: the secure core's queries for verified pages (#340): an
-- organisation's pages that may frame its embedded games, after the host's
-- yes. core/verified_page.go has checked that the caller may.

-- name: V2VerifiedPages :many
-- An organisation's verified pages, or one of them (id set).
SELECT id, institution_id, page_id, origin, page_url, callback_url, created_by, created_at, modified_at
FROM verified_page
WHERE institution_id = @institution_id::uuid
  AND (sqlc.narg(id)::uuid IS NULL OR id = sqlc.narg(id)::uuid)
ORDER BY created_at, id;

-- name: V2VerifiedPageOrganisation :one
-- The organisation a verified page belongs to.
SELECT institution_id FROM verified_page WHERE id = @id::uuid;

-- name: V2CreateVerifiedPage :one
INSERT INTO verified_page (institution_id, page_id, origin, page_url, callback_url, created_by)
VALUES (@institution_id::uuid, @page_id, @origin, @page_url, @callback_url, @created_by::uuid)
RETURNING id;

-- name: V2UpdateVerifiedPage :exec
UPDATE verified_page
SET origin = @origin, page_url = @page_url, callback_url = @callback_url, modified_at = now()
WHERE id = @id::uuid;

-- name: V2DeleteVerifiedPage :exec
-- Removes a verified page; the embed links set to it start nothing until
-- they get another (verified_page_id goes NULL).
DELETE FROM verified_page WHERE id = @id::uuid;
