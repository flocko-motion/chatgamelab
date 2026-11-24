-- A game's voices: for each voice its genre speaks in, the engine's
-- voice slots such as an Adventure's narrator, the game's choice: male,
-- female, any, or custom where the genre makes a voice of its own. A voice
-- it names nothing for is any. Which slots there are is the engine's
-- business, so a map: a genre that changes its slots needs no migration.
ALTER TABLE game ADD COLUMN voices jsonb NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(voices) = 'object');

-- An NPC-Live game with a voice style speaks in a voice made from it;
-- every other game speaks in a built-in voice its sessions cast.
UPDATE game SET voices = '{"character": "custom"}'::jsonb
WHERE genre = 'npc-live' AND btrim(voice_style) <> '';
