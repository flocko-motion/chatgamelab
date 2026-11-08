-- Persist the human-readable reasoning trace of how the active youth-protection
-- constraint was decided (db.ResolveConstraint) per game message, so the AI insights
-- view can explain *why* a constraint applied — alongside the existing snapshot of
-- source/text/source_name (migrations 028, 030).

ALTER TABLE game_session_message ADD COLUMN prompt_constraint_reasoning text NULL;
