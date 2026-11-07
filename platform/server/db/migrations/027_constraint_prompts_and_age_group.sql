-- Multi-level constraint prompts and user age groups.
--
-- Adds site-level age-based constraint prompts (U13 for 13-17, U18 for 18+),
-- organization-level constraint prompts, user age groups, and persists the
-- resolved constraint on game sessions.

-- Site-level age-based constraint prompts
ALTER TABLE system_settings ADD COLUMN prompt_constraint_u13 text NULL;
ALTER TABLE system_settings ADD COLUMN prompt_constraint_u18 text NULL;

-- Organization-level constraint prompt
ALTER TABLE institution ADD COLUMN prompt_constraints text NULL;

-- User age group: 'u13' (13-17 years), 'u18' (18+ years), NULL for guests
ALTER TABLE app_user ADD COLUMN age_group text NULL;
ALTER TABLE app_user ADD CONSTRAINT app_user_age_group_chk CHECK (age_group IS NULL OR age_group IN ('u13', 'u18'));

