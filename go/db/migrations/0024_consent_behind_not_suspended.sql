-- A newer declaration no longer suspends an organisation (#619). It stays
-- suspended while its newest entry in the consent log is no consent at all:
-- none yet, or a withdrawal. A consent to an older version leaves it active;
-- the admin sees it as behind and talks to it or suspends it.

CREATE OR REPLACE FUNCTION institution_consent_lacking(institution uuid)
RETURNS boolean AS $$
    SELECT NOT COALESCE((
        SELECT c.kind = 'consent'
        FROM institution_consent c
        WHERE c.institution_id = institution
        ORDER BY c.created_at DESC
        LIMIT 1
    ), false);
$$ LANGUAGE sql STABLE;

DROP FUNCTION institution_consent_missing(uuid, text, text);
