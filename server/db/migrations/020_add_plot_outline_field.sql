-- Migration 020: Add plot outline field
-- Stores the AI-generated plot outline separately from the message text.
-- The plot outline is the basis for prose generation (ExpandStory), not the final message.

ALTER TABLE game_session_message ADD COLUMN plot text NULL;
