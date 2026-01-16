-- Migration: Add show_ai_model_selector to app_user table
-- This allows users to configure whether they want to select AI models per session

ALTER TABLE app_user 
ADD COLUMN show_ai_model_selector boolean NOT NULL DEFAULT false;
