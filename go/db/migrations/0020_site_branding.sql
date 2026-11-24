-- The site's branding (#573): its name, operator, support address and the
-- links the web app shows. Only what the admin changed is stored; the core
-- fills every other field with its default.

ALTER TABLE system_settings ADD COLUMN branding jsonb NOT NULL DEFAULT '{}'::jsonb;
