-- Migration 011: Guest User Share Link
-- Links anonymous guest users to the game whose private share link created them.
-- Enables cleanup of guest users + sessions when a share link is revoked.
-- NULL for non-guest users.

ALTER TABLE app_user ADD COLUMN private_share_game_id uuid NULL REFERENCES game(id);
