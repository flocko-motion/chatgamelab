-- A picture's quality is the admin's to assign per tier (#589): empty keeps
-- the slot's own.
ALTER TABLE engine_model_assignment ADD COLUMN quality text NOT NULL DEFAULT '';
