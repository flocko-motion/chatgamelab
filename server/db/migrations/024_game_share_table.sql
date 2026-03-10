-- New game_share table (replaces game.private_share_* columns)
-- Enables multiple share links per game (personal + workshop) with metadata.
CREATE TABLE game_share (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id             uuid NOT NULL REFERENCES game(id),
    token               text NOT NULL UNIQUE,
    api_key_share_id    uuid NOT NULL REFERENCES api_key_share(id),
    institution_id      uuid NULL REFERENCES institution(id),
    workshop_id         uuid NULL REFERENCES workshop(id),
    remaining           integer NULL,  -- NULL = unlimited, 0 = exhausted
    created_by          uuid NULL REFERENCES app_user(id),
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- Add workshop setting to control whether participants can share games
ALTER TABLE workshop ADD COLUMN allow_game_sharing boolean NOT NULL DEFAULT false;

-- Drop existing private share links (clean slate — migration clears stale data)
UPDATE game SET
    private_share_hash = NULL,
    private_sponsored_api_key_share_id = NULL,
    private_share_remaining = NULL;

-- Repoint guest users from game to game_share
ALTER TABLE app_user ADD COLUMN private_share_id uuid NULL REFERENCES game_share(id);
-- Clear existing guest user links (shares are gone)
UPDATE app_user SET private_share_game_id = NULL WHERE private_share_game_id IS NOT NULL;
-- Drop old FK and column
ALTER TABLE app_user DROP CONSTRAINT IF EXISTS app_user_private_share_game_fk;
ALTER TABLE app_user DROP COLUMN private_share_game_id;

-- Remove old columns from game
ALTER TABLE game DROP COLUMN private_share_hash;
ALTER TABLE game DROP COLUMN private_sponsored_api_key_share_id;
ALTER TABLE game DROP COLUMN private_share_remaining;
