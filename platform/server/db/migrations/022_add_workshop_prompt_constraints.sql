-- Migration 021: Add workshop prompt constraints
-- Allows workshop admins to define extra prompt constraints injected into session system prompts.

ALTER TABLE workshop ADD COLUMN prompt_constraints text NULL;
