-- Track which constraint source was active when a game message was generated.
-- Source labels: "workshop", "organisation", "site13", "site13p", "site18".
--   site13  = 13-17, no parental consent
--   site13p = 13-17, with parental consent (added in migration 029)
--   site18  = 18+
-- Used to surface the active constraint rule in the player's session details modal.

ALTER TABLE game_session_message ADD COLUMN prompt_constraint_source text NULL;
