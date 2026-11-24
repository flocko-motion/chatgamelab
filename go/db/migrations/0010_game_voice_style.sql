-- A game's voice style: how its voice sounds, in the designer's words, as the
-- image style says how its pictures look. A genre that speaks reads it, such
-- as NPC-Live for its character's voice; empty leaves the voice to the
-- engine.
ALTER TABLE game ADD COLUMN voice_style text NOT NULL DEFAULT '';
