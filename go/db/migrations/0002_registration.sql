-- An account exists from its first sign-in; it has a name once registration is
-- complete, and is unusable until then.
ALTER TABLE app_user ALTER COLUMN name DROP NOT NULL;
