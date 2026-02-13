-- Migration 018: Add language column to game_session
-- Stores the user's language preference at session creation time (ISO 639-1 code).
-- Defaults to 'en' for existing sessions.
-- Uses IF NOT EXISTS because this column may already exist from a prior migration.

ALTER TABLE game_session ADD COLUMN IF NOT EXISTS language text NOT NULL DEFAULT 'en';
