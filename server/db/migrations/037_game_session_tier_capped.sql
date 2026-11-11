-- A session asked for a higher AI quality tier than the free-use key paying for it
-- is released up to: the tier it asked for, and who released the key (organization/server).
-- Both stay NULL while the session plays at the tier it asked for.
ALTER TABLE game_session ADD COLUMN ai_quality_tier_requested text NULL;
ALTER TABLE game_session ADD COLUMN ai_quality_tier_limited_by text NULL;
