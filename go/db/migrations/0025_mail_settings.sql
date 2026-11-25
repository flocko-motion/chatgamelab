-- How the site sends mail (#621): the admin sets the mail server in the
-- site's settings. An empty host only logs mails. The password is written,
-- never read back to anyone. consent_copy_email receives a copy of every
-- organisation's consent; empty: the site's support address.

ALTER TABLE system_settings
    ADD COLUMN smtp_host text NOT NULL DEFAULT '',
    ADD COLUMN smtp_port integer NOT NULL DEFAULT 587 CHECK (smtp_port BETWEEN 1 AND 65535),
    ADD COLUMN smtp_user text NOT NULL DEFAULT '',
    ADD COLUMN smtp_password text NOT NULL DEFAULT '',
    ADD COLUMN smtp_from text NOT NULL DEFAULT '',
    ADD COLUMN consent_copy_email text NOT NULL DEFAULT '';
