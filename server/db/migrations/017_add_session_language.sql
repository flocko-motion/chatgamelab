-- Migration 017: Add language column to game_session
-- Stores the user's language preference at session creation time (ISO 639-1 code).
-- Defaults to 'en' for existing sessions.

ALTER TABLE game_session ADD COLUMN language text NOT NULL DEFAULT 'en';
