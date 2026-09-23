-- The public workshop page at /w/<public_slug>, visible while workshop.public is on.
-- Existing workshops receive their slug from the backend at startup, because the
-- word list lives in Go (db.BackfillWorkshopPublicSlugs).
ALTER TABLE workshop ADD COLUMN public_slug text NULL UNIQUE;
ALTER TABLE workshop ADD COLUMN public_description text NULL;

-- Share links the public page creates for itself, one per game and workshop. They are
-- kept apart from the links leaders create by hand, so neither reuses the other.
ALTER TABLE game_share ADD COLUMN public_page boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX game_share_public_page_uniq ON game_share (game_id, workshop_id) WHERE public_page;
