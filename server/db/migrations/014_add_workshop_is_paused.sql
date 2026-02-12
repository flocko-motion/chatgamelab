-- Migration 014: Add is_paused to workshop
-- Controls whether the workshop is paused (participants cannot play while paused).
-- Default false: workshop is not paused.

ALTER TABLE workshop ADD COLUMN is_paused boolean NOT NULL DEFAULT false;
