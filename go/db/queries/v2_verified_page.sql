-- v2: the secure core's queries for verified pages (#340): an
-- organisation's one page that may frame its embedded games, after an
-- admin's approval and the host's yes. core/verified_page.go has checked
-- that the caller may.

-- name: V2VerifiedPages :many
-- Verified pages: one organisation's (institution_id set), or every one;
-- one of them (id set).
SELECT v.id, v.institution_id, i.name AS institution_name, v.page_id, v.origin, v.page_url, v.callback_url,
  v.approved_at, v.created_by, v.created_at, v.modified_at
FROM verified_page v
JOIN institution i ON i.id = v.institution_id
WHERE (sqlc.narg(institution_id)::uuid IS NULL OR v.institution_id = sqlc.narg(institution_id)::uuid)
  AND (sqlc.narg(id)::uuid IS NULL OR v.id = sqlc.narg(id)::uuid)
ORDER BY v.approved_at IS NOT NULL, v.created_at, v.id;

-- name: V2VerifiedPageOrganisation :one
-- The organisation a verified page belongs to.
SELECT institution_id FROM verified_page WHERE id = @id::uuid;

-- name: V2CreateVerifiedPage :one
-- An organisation's verified page, unapproved; verified_page_institution_uniq
-- keeps it one per organisation.
INSERT INTO verified_page (institution_id, page_id, origin, page_url, callback_url, created_by)
VALUES (@institution_id::uuid, @page_id, @origin, @page_url, @callback_url, @created_by::uuid)
ON CONFLICT (institution_id) DO NOTHING
RETURNING id;

-- name: V2UpdateVerifiedPage :exec
-- Changes a verified page's address and callback; a change withdraws its
-- approval.
UPDATE verified_page
SET origin = @origin, page_url = @page_url, callback_url = @callback_url, modified_at = now(),
    approved_at = CASE WHEN (origin, page_url, callback_url) IS DISTINCT FROM (@origin::text, @page_url::text, @callback_url::text)
                       THEN NULL ELSE approved_at END,
    approved_by = CASE WHEN (origin, page_url, callback_url) IS DISTINCT FROM (@origin::text, @page_url::text, @callback_url::text)
                       THEN NULL ELSE approved_by END
WHERE id = @id::uuid;

-- name: V2ApproveVerifiedPage :exec
-- An admin approves a verified page (approved_by set) or withdraws it (NULL).
UPDATE verified_page
SET approved_at = CASE WHEN sqlc.narg(approved_by)::uuid IS NULL THEN NULL ELSE now() END,
    approved_by = sqlc.narg(approved_by)::uuid
WHERE id = @id::uuid;

-- name: V2DeleteVerifiedPage :exec
-- Removes a verified page; its organisation's embedded games start nothing
-- until it has another.
DELETE FROM verified_page WHERE id = @id::uuid;
