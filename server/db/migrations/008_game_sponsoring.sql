-- Migration 008: Game Sponsoring
-- Adds game_id to api_key_share for game-scoped sponsoring.
-- Renames public_sponsored_api_key_id -> public_sponsored_api_key_share_id on game table
-- (now references api_key_share instead of api_key directly).

-- 1. Add game_id column to api_key_share
ALTER TABLE api_key_share ADD COLUMN game_id uuid NULL REFERENCES game(id);

-- 2. Update the target check constraint to allow game_id as a valid target
ALTER TABLE api_key_share DROP CONSTRAINT api_key_share_target_chk;
ALTER TABLE api_key_share ADD CONSTRAINT api_key_share_target_chk CHECK (
    user_id IS NOT NULL OR workshop_id IS NOT NULL OR institution_id IS NOT NULL OR game_id IS NOT NULL
);

-- 3. Rename public_sponsored_api_key_id -> public_sponsored_api_key_share_id on game
--    Drop the old FK (references api_key), rename, add new FK (references api_key_share)
ALTER TABLE game DROP CONSTRAINT IF EXISTS game_public_sponsored_api_key_id_fkey;
ALTER TABLE game RENAME COLUMN public_sponsored_api_key_id TO public_sponsored_api_key_share_id;
ALTER TABLE game ADD CONSTRAINT game_public_sponsored_api_key_share_id_fkey
    FOREIGN KEY (public_sponsored_api_key_share_id) REFERENCES api_key_share(id);

-- 4. Similarly rename private_sponsored_api_key_id -> private_sponsored_api_key_share_id
ALTER TABLE game DROP CONSTRAINT IF EXISTS game_private_sponsored_api_key_id_fkey;
ALTER TABLE game RENAME COLUMN private_sponsored_api_key_id TO private_sponsored_api_key_share_id;
ALTER TABLE game ADD CONSTRAINT game_private_sponsored_api_key_share_id_fkey
    FOREIGN KEY (private_sponsored_api_key_share_id) REFERENCES api_key_share(id);
