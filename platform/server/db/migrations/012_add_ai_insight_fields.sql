-- Migration 012: Add AI Insight Fields
-- Stores raw AI request/response data on game session messages for educational debug view.
-- These fields allow users to inspect what prompts were sent to the AI and what it returned.

ALTER TABLE game_session_message ADD COLUMN prompt_status_update text NULL;
ALTER TABLE game_session_message ADD COLUMN prompt_response_schema text NULL;
ALTER TABLE game_session_message ADD COLUMN prompt_image_generation text NULL;
ALTER TABLE game_session_message ADD COLUMN prompt_expand_story text NULL;
ALTER TABLE game_session_message ADD COLUMN response_raw text NULL;
ALTER TABLE game_session_message ADD COLUMN token_usage jsonb NULL;
ALTER TABLE game_session_message ADD COLUMN url_analytics text NULL;
