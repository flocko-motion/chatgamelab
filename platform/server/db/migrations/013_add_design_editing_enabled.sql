-- Migration 013: Add design_editing_enabled to workshop
-- Controls whether workshop participants can edit game design (theme).
-- Default false: participants cannot edit design unless explicitly enabled by staff/head.

ALTER TABLE workshop ADD COLUMN design_editing_enabled boolean NOT NULL DEFAULT false;
