-- Track which constraint source was active when a game message was generated.
-- Source labels: "workshop", "organisation", "site13", "site18".
-- Used to surface the active constraint rule in the player's session details modal.

ALTER TABLE game_session_message ADD COLUMN prompt_constraint_source text NULL;
