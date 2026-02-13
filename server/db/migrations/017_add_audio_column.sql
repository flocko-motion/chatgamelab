-- Migration 017: Add audio column to game_session_message
-- The audio column was missing from the dev database but exists in schema.sql
-- This migration adds the audio bytea column to store audio narration data

ALTER TABLE game_session_message ADD COLUMN audio bytea NULL;
