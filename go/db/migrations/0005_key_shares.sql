-- Key shares follow the lean sharing model (#451): a share is permission to
-- use a key. Its owner uses their key themselves (the self-share) and gives
-- it to their own organisation; money leaves the organisation only through a
-- sponsored game or a share link, which records the share it draws from.
--
--  * api_key_share gets source_share_id: the share a sponsorship or a share
--    link draws from; NULL, the key itself. Taking a share back stops what
--    draws from it (core/key*.go).
--  * workshop.default_api_key_share_id and institution.free_use_api_key_share_id
--    follow their share: gone, they are NULL, and the workshop does not play.
--  * v1's shares the model has no place for go: to other users, to
--    workshops, and to an organisation the key's owner is not a member of;
--    a workshop or an organisation pointing at anything but one of its
--    organisation's shares loses that pointer.

ALTER TABLE api_key_share ADD COLUMN IF NOT EXISTS source_share_id uuid NULL REFERENCES api_key_share(id);

-- v1 let one row name several receivers; an organisation share keeps only that
UPDATE api_key_share SET user_id = NULL, workshop_id = NULL
WHERE institution_id IS NOT NULL AND game_id IS NULL
  AND (user_id IS NOT NULL OR workshop_id IS NOT NULL);

-- the shares the model has no place for
DROP TABLE IF EXISTS dropped_share;
CREATE TABLE dropped_share AS
SELECT s.id
FROM api_key_share s
JOIN api_key k ON k.id = s.api_key_id
WHERE s.game_id IS NULL AND (
      -- to a workshop
      (s.institution_id IS NULL AND s.workshop_id IS NOT NULL)
      -- to another user
   OR (s.institution_id IS NULL AND s.workshop_id IS NULL AND s.user_id IS DISTINCT FROM k.user_id)
      -- to an organisation the owner is not a member of
   OR (s.institution_id IS NOT NULL AND NOT EXISTS (
         SELECT 1 FROM user_role r
         WHERE r.user_id = k.user_id AND r.institution_id = s.institution_id
           AND r.role IN ('head', 'staff', 'individual')))
);

UPDATE app_user      SET default_api_key_share_id = NULL          WHERE default_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE workshop      SET default_api_key_share_id = NULL          WHERE default_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE institution   SET free_use_api_key_share_id = NULL         WHERE free_use_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE game          SET public_sponsored_api_key_share_id = NULL WHERE public_sponsored_api_key_share_id IN (SELECT id FROM dropped_share);
UPDATE game_session  SET api_key_share_id = NULL                  WHERE api_key_share_id IN (SELECT id FROM dropped_share);
DELETE FROM api_key_share WHERE id IN (SELECT id FROM dropped_share);
DROP TABLE dropped_share;

-- a workshop plays with one of its organisation's shares, an organisation's
-- free use is one of its own; anything else goes
UPDATE workshop w SET default_api_key_share_id = NULL
WHERE default_api_key_share_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM api_key_share s
  WHERE s.id = w.default_api_key_share_id AND s.institution_id = w.institution_id AND s.game_id IS NULL);
UPDATE institution i SET free_use_api_key_share_id = NULL
WHERE free_use_api_key_share_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM api_key_share s
  WHERE s.id = i.free_use_api_key_share_id AND s.institution_id = i.id AND s.game_id IS NULL);

-- both pointers follow their share
ALTER TABLE workshop DROP CONSTRAINT IF EXISTS workshop_default_api_key_share_fk;
ALTER TABLE workshop ADD CONSTRAINT workshop_default_api_key_share_fk
    FOREIGN KEY (default_api_key_share_id) REFERENCES api_key_share(id) ON DELETE SET NULL;
ALTER TABLE institution DROP CONSTRAINT IF EXISTS institution_free_use_api_key_share_fk;
ALTER TABLE institution ADD CONSTRAINT institution_free_use_api_key_share_fk
    FOREIGN KEY (free_use_api_key_share_id) REFERENCES api_key_share(id) ON DELETE SET NULL;
