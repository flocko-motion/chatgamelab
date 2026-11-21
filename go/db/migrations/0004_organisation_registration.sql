-- Organisations register themselves (ported from v1, migration 038).
--
--  * institution gets its operator (Träger), its address and an admin's
--    suspension
--  * institution_consent is the append-only log of every consent to the
--    declaration and of every withdrawal: the proof, kept even when
--    organisation or account are gone
--
-- An organisation is suspended while an admin suspended it, or while its
-- newest entry in the log is no consent to the current declaration.
--
-- A database v1 migrated holds a version of this already; every statement
-- brings it to this shape.

ALTER TABLE institution ADD COLUMN IF NOT EXISTS operator text NULL;
ALTER TABLE institution ADD COLUMN IF NOT EXISTS street text NULL;
ALTER TABLE institution ADD COLUMN IF NOT EXISTS postal_code text NULL;
ALTER TABLE institution ADD COLUMN IF NOT EXISTS city text NULL;

-- v1 called an admin's suspension a lock (disabled_at, disabled_reason)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'institution' AND column_name = 'disabled_at') THEN
        ALTER TABLE institution RENAME COLUMN disabled_at TO suspended_at;
        ALTER TABLE institution RENAME COLUMN disabled_reason TO suspended_reason;
    END IF;
END $$;
ALTER TABLE institution ADD COLUMN IF NOT EXISTS suspended_at timestamptz NULL;
ALTER TABLE institution ADD COLUMN IF NOT EXISTS suspended_reason text NULL;

-- v1's registration applications: a confirmed one has its consent in the log,
-- one never confirmed never became an organisation
DROP TABLE IF EXISTS institution_application;

-- No foreign keys: the proof must survive deleting the organisation or the account.
CREATE TABLE IF NOT EXISTS institution_consent (
    id                      uuid PRIMARY KEY,
    created_at              timestamptz NOT NULL DEFAULT now(),
    institution_id          uuid NOT NULL,
    user_id                 uuid NOT NULL,
    -- 'consent': user_id agreed to the declaration in the organisation's
    -- name; 'withdrawal': user_id withdrew the consent before it, whose
    -- declaration version and hash it repeats
    kind                    text NOT NULL DEFAULT 'consent',
    email                   text NULL,
    person_name             text NULL,
    person_function         text NULL,
    institution_name        text NOT NULL,
    institution_operator    text NULL,
    street                  text NULL,
    postal_code             text NULL,
    city                    text NULL,
    declaration_version     text NOT NULL,
    declaration_hash        text NOT NULL
);

-- v1's log: consents only, each tied to its application
ALTER TABLE institution_consent ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'consent';
ALTER TABLE institution_consent DROP COLUMN IF EXISTS application_id;
ALTER TABLE institution_consent ALTER COLUMN email DROP NOT NULL;
ALTER TABLE institution_consent ALTER COLUMN person_name DROP NOT NULL;
ALTER TABLE institution_consent ALTER COLUMN person_function DROP NOT NULL;
ALTER TABLE institution_consent ALTER COLUMN institution_operator DROP NOT NULL;
ALTER TABLE institution_consent ALTER COLUMN street DROP NOT NULL;
ALTER TABLE institution_consent ALTER COLUMN postal_code DROP NOT NULL;
ALTER TABLE institution_consent ALTER COLUMN city DROP NOT NULL;

-- a consent names who agreed and the organisation's details; a withdrawal
-- names only who withdrew
ALTER TABLE institution_consent DROP CONSTRAINT IF EXISTS institution_consent_kind_chk;
ALTER TABLE institution_consent ADD CONSTRAINT institution_consent_kind_chk CHECK (
    kind = 'withdrawal'
    OR (kind = 'consent'
        AND email IS NOT NULL AND person_name IS NOT NULL AND person_function IS NOT NULL
        AND institution_operator IS NOT NULL AND street IS NOT NULL
        AND postal_code IS NOT NULL AND city IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS institution_consent_institution_idx ON institution_consent (institution_id, created_at DESC);

CREATE OR REPLACE FUNCTION institution_consent_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'institution_consent is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS institution_consent_no_update_delete ON institution_consent;
CREATE TRIGGER institution_consent_no_update_delete
    BEFORE UPDATE OR DELETE ON institution_consent
    FOR EACH ROW EXECUTE FUNCTION institution_consent_immutable();

-- Whether the organisation's newest entry in the consent log is no consent to
-- the declaration in version and hash: none yet, a withdrawal, or a consent
-- to another version. Its consent cause of being suspended, in one place.
CREATE OR REPLACE FUNCTION institution_consent_missing(institution uuid, version text, hash text)
RETURNS boolean AS $$
    SELECT NOT COALESCE((
        SELECT c.kind = 'consent' AND c.declaration_version = version AND c.declaration_hash = hash
        FROM institution_consent c
        WHERE c.institution_id = institution
        ORDER BY c.created_at DESC
        LIMIT 1
    ), false);
$$ LANGUAGE sql STABLE;
