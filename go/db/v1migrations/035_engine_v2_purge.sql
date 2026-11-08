-- Migration 035: the engine runs sessions
-- Sessions and their messages leave the platform; the engine's session tables
-- come with its integration.
-- game.theme holds the name of the engine theme; null means the engine's default.
-- Workshops no longer gate design editing.

DROP TABLE IF EXISTS game_session_message;
DROP TABLE IF EXISTS game_session;

ALTER TABLE game ALTER COLUMN theme TYPE text USING NULLIF(theme->>'preset', '');

ALTER TABLE workshop DROP COLUMN IF EXISTS design_editing_enabled;
