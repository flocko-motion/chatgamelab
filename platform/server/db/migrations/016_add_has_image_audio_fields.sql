-- Migration 016: Add HasImage/HasAudio capability flags
-- Set at message creation time based on the platform's tier capabilities.
-- Tells the frontend whether to expect image/audio for this message.

ALTER TABLE game_session_message ADD COLUMN has_image boolean NOT NULL DEFAULT false;
ALTER TABLE game_session_message ADD COLUMN has_audio boolean NOT NULL DEFAULT false;
