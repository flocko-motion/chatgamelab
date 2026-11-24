-- A verified page takes an admin's approval, and an organisation has one
-- (#340). Until a page is approved, and again after its address or callback
-- changes, its games start nothing and no site frames them.

-- One verified page per organisation: the oldest stays.
DELETE FROM verified_page v
WHERE EXISTS (SELECT 1 FROM verified_page o
              WHERE o.institution_id = v.institution_id AND (o.created_at, o.id) < (v.created_at, v.id));
DROP INDEX IF EXISTS verified_page_institution;
CREATE UNIQUE INDEX verified_page_institution_uniq ON verified_page (institution_id);

ALTER TABLE verified_page
    ADD COLUMN approved_at timestamptz NULL,
    ADD COLUMN approved_by uuid NULL REFERENCES app_user(id) ON DELETE SET NULL;
