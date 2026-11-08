-- Split the 13-17 age bucket into two consent-based cohorts.
--
-- Before: age_group IN ('u13', 'u18') — 'u13' = 13-17, 'u18' = 18+.
-- After:  age_group IN ('u13', 'u13p', 'u18')
--   'u13'  = 13-17, NO parental consent on file (strictest; also the safe default when age is unknown).
--   'u13p' = 13-17, WITH parental consent on file.
--   'u18'  = 18+.
-- Existing 'u13' rows retain their meaning (no-consent), which is the safer bucket.
--
-- Also adds a site-level constraint prompt column for the new u13p cohort,
-- mirroring prompt_constraint_u13 / prompt_constraint_u18 from migration 027.

ALTER TABLE system_settings ADD COLUMN prompt_constraint_u13p text NULL;

ALTER TABLE app_user DROP CONSTRAINT app_user_age_group_chk;
ALTER TABLE app_user ADD CONSTRAINT app_user_age_group_chk CHECK (age_group IS NULL OR age_group IN ('u13', 'u13p', 'u18'));
