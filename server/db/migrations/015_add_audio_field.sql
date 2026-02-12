-- Migration 015: Add Audio Field
-- Stores generated TTS audio data on game session messages (parallel to image field).
-- Used when the "max" quality tier is selected (OpenAI only).

ALTER TABLE game_session_message ADD COLUMN audio bytea NULL;
