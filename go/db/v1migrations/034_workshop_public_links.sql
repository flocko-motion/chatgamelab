-- Further reading a workshop offers on its public page /w/<public_slug>:
-- a JSON array of {"title","description","url"}, validated in Go
-- (db.ValidatePublicLinks) because only http and https may be published.
ALTER TABLE workshop ADD COLUMN public_links jsonb NULL;
