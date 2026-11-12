-- Organisation self-registration:
--  * institution gets operator (Träger), address, and a reversible admin lock
--  * institution_application holds a registration/upgrade request until the e-mail is confirmed
--  * institution_consent is the immutable log of every declaration consent

ALTER TABLE institution ADD COLUMN operator text NULL;
ALTER TABLE institution ADD COLUMN street text NULL;
ALTER TABLE institution ADD COLUMN postal_code text NULL;
ALTER TABLE institution ADD COLUMN city text NULL;
ALTER TABLE institution ADD COLUMN disabled_at timestamptz NULL;
ALTER TABLE institution ADD COLUMN disabled_reason text NULL;

CREATE TABLE institution_application (
    id                      uuid PRIMARY KEY,
    created_at              timestamptz NOT NULL DEFAULT now(),
    modified_at             timestamptz NOT NULL DEFAULT now(),
    -- Personal data (user_id, person_*, email) is set to NULL by the retention job
    -- for applications that were never confirmed; the rest stays for the admin overview.
    user_id                 uuid NULL REFERENCES app_user(id) ON DELETE SET NULL,
    kind                    text NOT NULL,
    person_name             text NULL,
    person_function         text NULL,
    email                   text NULL,
    institution_name        text NOT NULL,
    institution_operator    text NOT NULL,
    street                  text NOT NULL,
    postal_code             text NOT NULL,
    city                    text NOT NULL,
    prompt_constraint       text NOT NULL,
    declaration_version     text NOT NULL,
    declaration_hash        text NOT NULL,
    token_hash              text NOT NULL UNIQUE,
    expires_at              timestamptz NOT NULL,
    status                  text NOT NULL DEFAULT 'pending',
    confirmed_at            timestamptz NULL,
    institution_id          uuid NULL REFERENCES institution(id) ON DELETE SET NULL,
    anonymized_at           timestamptz NULL,
    CONSTRAINT institution_application_kind_chk CHECK (kind IN ('register', 'upgrade')),
    CONSTRAINT institution_application_status_chk CHECK (status IN ('pending', 'confirmed', 'expired'))
);

CREATE INDEX institution_application_user_idx ON institution_application (user_id);
CREATE INDEX institution_application_status_idx ON institution_application (status, created_at DESC);

-- No foreign keys on purpose: the proof must survive deletion of the organisation or account.
CREATE TABLE institution_consent (
    id                      uuid PRIMARY KEY,
    created_at              timestamptz NOT NULL DEFAULT now(),
    institution_id          uuid NOT NULL,
    user_id                 uuid NOT NULL,
    application_id          uuid NULL,
    email                   text NOT NULL,
    person_name             text NOT NULL,
    person_function         text NOT NULL,
    institution_name        text NOT NULL,
    institution_operator    text NOT NULL,
    street                  text NOT NULL,
    postal_code             text NOT NULL,
    city                    text NOT NULL,
    declaration_version     text NOT NULL,
    declaration_hash        text NOT NULL
);

CREATE INDEX institution_consent_institution_idx ON institution_consent (institution_id, created_at DESC);

CREATE FUNCTION institution_consent_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'institution_consent is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER institution_consent_no_update_delete
    BEFORE UPDATE OR DELETE ON institution_consent
    FOR EACH ROW EXECUTE FUNCTION institution_consent_immutable();
