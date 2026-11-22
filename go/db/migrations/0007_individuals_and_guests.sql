-- The role says where someone is (spec, Roles). Head and staff are in an
-- organisation; an individual is in none. Taking part in a workshop makes no
-- one part of its organisation: a guest belongs to their workshop, which
-- belongs to the organisation, and an individual taking part keeps their
-- role. So a role names an organisation for head and staff only.
--
-- v1 let an organisation invite as "individual", and wrote every guest's
-- role with the organisation. Here:
--  * an individual of an organisation is detached: an individual, in no
--    organisation. Their keys' shares to that organisation go, with what
--    draws from them, as taking a share back ends them (core/key.go,
--    endShares): sponsorships, and share links with their guests, whose
--    sessions go with them;
--  * a guest's role drops the organisation; it is their workshop's;
--  * a constraint keeps it so.

DROP TABLE IF EXISTS dropped_share;
CREATE TABLE dropped_share AS
SELECT s.id
FROM api_key_share s
JOIN api_key k ON k.id = s.api_key_id
WHERE s.institution_id IS NOT NULL AND s.game_id IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM user_role r
    WHERE r.user_id = k.user_id AND r.institution_id = s.institution_id
      AND r.role IN ('head', 'staff'));
-- and what draws from them
INSERT INTO dropped_share
SELECT s.id FROM api_key_share s WHERE s.source_share_id IN (SELECT id FROM dropped_share);

-- the share links they pay for end, with their guests (sessions cascade)
DROP TABLE IF EXISTS dropped_link;
CREATE TABLE dropped_link AS
SELECT id FROM game_share WHERE api_key_share_id IN (SELECT id FROM dropped_share);
DELETE FROM user_role           WHERE user_id IN (SELECT id FROM app_user WHERE private_share_id IN (SELECT id FROM dropped_link));
DELETE FROM user_favourite_game WHERE user_id IN (SELECT id FROM app_user WHERE private_share_id IN (SELECT id FROM dropped_link));
DELETE FROM app_user  WHERE private_share_id IN (SELECT id FROM dropped_link);
DELETE FROM game_share WHERE id IN (SELECT id FROM dropped_link);
DROP TABLE dropped_link;

UPDATE app_user      SET default_api_key_share_id = NULL          WHERE default_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE workshop      SET default_api_key_share_id = NULL          WHERE default_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE institution   SET free_use_api_key_share_id = NULL         WHERE free_use_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE game          SET public_sponsored_api_key_share_id = NULL WHERE public_sponsored_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE game_session  SET api_key_share_id = NULL                  WHERE api_key_share_id IN (SELECT id FROM dropped_share);
-- drawn shares first: they reference the ones they draw from
DELETE FROM api_key_share WHERE id IN (SELECT id FROM dropped_share) AND source_share_id IS NOT NULL;
DELETE FROM api_key_share WHERE id IN (SELECT id FROM dropped_share);
DROP TABLE dropped_share;

-- individuals are in no organisation; guests are in their workshop
UPDATE user_role SET institution_id = NULL WHERE role IN ('individual', 'guest') AND institution_id IS NOT NULL;

ALTER TABLE user_role DROP CONSTRAINT IF EXISTS user_role_organisation_chk;
ALTER TABLE user_role ADD CONSTRAINT user_role_organisation_chk
    CHECK (institution_id IS NULL OR role IN ('head', 'staff'));
