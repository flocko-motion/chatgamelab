-- Migration 021: Add API key type to game session messages
-- Tracks which type of API key was used to generate each AI response.
-- Shown in the AI Insight panel so users know the key source (e.g. workshop, personal).

ALTER TABLE game_session_message ADD COLUMN api_key_type text NULL;

ALTER TABLE game_session_message ADD CONSTRAINT game_session_message_api_key_type_chk
  CHECK (api_key_type IS NULL OR api_key_type IN (
    'workshop', 'sponsor', 'institution_free_use',
    'personal', 'system_free_use', 'private_share'));
