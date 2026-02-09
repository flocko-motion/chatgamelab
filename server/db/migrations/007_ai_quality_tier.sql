-- Migration 007: AI Quality Tier
-- Replaces per-user/per-workshop model selection with a cascading ai_quality_tier system.
-- The tier (high/medium/low) is tied to the API key source, not chosen by the user.
-- All tier fields are nullable; NULL means "use server default" (system_settings.default_ai_quality_tier).

-- Workshop: rename use_specific_ai_model -> ai_quality_tier
ALTER TABLE workshop RENAME COLUMN use_specific_ai_model TO ai_quality_tier;

-- Workshop: remove show_ai_model_selector (users no longer choose)
ALTER TABLE workshop DROP COLUMN show_ai_model_selector;

-- Institution: add free_use_ai_quality_tier (tier for the institution free-use key)
ALTER TABLE institution ADD COLUMN free_use_ai_quality_tier text NULL;

-- User: add ai_quality_tier (tier for user's own default key)
ALTER TABLE app_user ADD COLUMN ai_quality_tier text NULL;

-- User: remove show_ai_model_selector (users no longer choose)
ALTER TABLE app_user DROP COLUMN show_ai_model_selector;

-- System settings: rename default_ai_model -> default_ai_quality_tier (the ultimate server-wide fallback)
ALTER TABLE system_settings RENAME COLUMN default_ai_model TO default_ai_quality_tier;
ALTER TABLE system_settings ALTER COLUMN default_ai_quality_tier SET DEFAULT 'medium';
UPDATE system_settings SET default_ai_quality_tier = 'medium' WHERE default_ai_quality_tier = '' OR default_ai_quality_tier IS NULL;

-- System settings: add free_use_ai_quality_tier (tier for the system free-use key, nullable)
ALTER TABLE system_settings ADD COLUMN free_use_ai_quality_tier text NULL;
