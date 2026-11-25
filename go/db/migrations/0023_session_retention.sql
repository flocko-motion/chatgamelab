-- How long the site keeps a session not played, in hours (#556): the admin
-- sets it, the janitor reads it on every pass. 0 keeps every session.

ALTER TABLE system_settings
    ADD COLUMN session_retention_hours integer NOT NULL DEFAULT 48
    CHECK (session_retention_hours >= 0);
