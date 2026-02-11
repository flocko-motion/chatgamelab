-- Migration 009: Game Theme
-- Adds optional theme (jsonb) column to game table.
-- When set, the game's theme is used directly instead of AI-generating one per session.

ALTER TABLE game ADD COLUMN theme jsonb NULL;
