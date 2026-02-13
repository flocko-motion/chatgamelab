-- Migration 015: Add ON DELETE SET NULL to game sponsor FK constraints
-- Fixes: "update or delete on table api_key_share violates foreign key constraint
-- game_private_sponsored_api_key_share_id_fkey on table game"
-- Without ON DELETE SET NULL, every code path that deletes an api_key_share row
-- must manually clear the game reference first. Adding ON DELETE SET NULL makes
-- PostgreSQL handle this automatically, eliminating the entire class of bugs.

ALTER TABLE game DROP CONSTRAINT IF EXISTS game_public_sponsored_api_key_share_id_fkey;
ALTER TABLE game ADD CONSTRAINT game_public_sponsored_api_key_share_id_fkey
    FOREIGN KEY (public_sponsored_api_key_share_id) REFERENCES api_key_share(id) ON DELETE SET NULL;

ALTER TABLE game DROP CONSTRAINT IF EXISTS game_private_sponsored_api_key_share_id_fkey;
ALTER TABLE game ADD CONSTRAINT game_private_sponsored_api_key_share_id_fkey
    FOREIGN KEY (private_sponsored_api_key_share_id) REFERENCES api_key_share(id) ON DELETE SET NULL;
