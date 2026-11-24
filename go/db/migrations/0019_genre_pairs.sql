-- Genres come in pairs (#517): a game's genre holds a live and a rounds half,
-- and the session picks one. A game that was a live character is now an NPC,
-- which plays live or in rounds; a game of two characters is a Discussion.
-- An adventure keeps its genre. Runs after 0018, which reads the old names.
UPDATE game SET genre = 'npc' WHERE genre = 'npc-live';
UPDATE game SET genre = 'discussion' WHERE genre = 'npc-duo';
