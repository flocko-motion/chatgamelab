-- Persist the resolved constraint text and a human-readable source name per game message,
-- so the AI insights view can show what was actually applied even if the underlying
-- workshop/org/site constraint or entity name changes later.
--
-- prompt_constraint_source (from migration 028) keeps the source label
--   ("workshop", "organisation", "site13", "site13p", "site18").
-- The new columns add the snapshot:
--   prompt_constraint_text         = the constraint text that went to the AI for this message.
--   prompt_constraint_source_name  = readable origin, e.g. 'Kreativ-AG (Hauptschule Lola)'
--                                    for a workshop, 'Hauptschule Lola' for an organisation,
--                                    NULL for site-by-age (the label is self-explanatory).

ALTER TABLE game_session_message ADD COLUMN prompt_constraint_text text NULL;
ALTER TABLE game_session_message ADD COLUMN prompt_constraint_source_name text NULL;
