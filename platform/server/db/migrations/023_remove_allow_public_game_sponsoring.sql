-- Remove allow_public_game_sponsoring flag from api_key_share.
-- Any share where the user has assignment rights can now be used for sponsoring.
ALTER TABLE api_key_share DROP COLUMN allow_public_game_sponsoring;
