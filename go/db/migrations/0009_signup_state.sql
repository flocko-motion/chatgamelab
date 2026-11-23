-- Sign-up is a state of the account (spec, Registration): from the first
-- sign-in until registration completes it. registered_at marks the
-- completion; registration sets it once and nothing resets it. A role is no
-- marker: roles change in normal use.
--
-- The default makes every account registered at creation; only the first
-- Auth0 sign-in (V2CreateAuth0Account) makes one in sign-up, with NULL.
ALTER TABLE app_user ADD COLUMN registered_at timestamptz NULL DEFAULT now();

-- Accounts so far: with a name, registered when they were made; without
-- one, still in sign-up.
UPDATE app_user SET registered_at = created_at WHERE COALESCE(name, '') <> '';
UPDATE app_user SET registered_at = NULL WHERE COALESCE(name, '') = '';
