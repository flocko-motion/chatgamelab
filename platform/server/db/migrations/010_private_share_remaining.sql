-- Migration 010: Private Share Remaining Counter
-- Adds a session counter for private share links.
-- NULL = unlimited, >0 = can play, 0 = exhausted.

ALTER TABLE game ADD COLUMN private_share_remaining integer NULL;
