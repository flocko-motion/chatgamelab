-- A verified page is approved for its site, scheme and host (#340): any
-- port and any path on it may frame its games and answer as its callback.
-- An approval stays as long as the site does.

ALTER TABLE verified_page RENAME COLUMN origin TO site;

-- An origin's port goes: http://localhost:8000 is the site http://localhost.
UPDATE verified_page SET site = regexp_replace(site, ':[0-9]+$', '');
