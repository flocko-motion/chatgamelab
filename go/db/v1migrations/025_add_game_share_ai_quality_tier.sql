-- Add AI quality tier to game shares.
-- For non-workshop shares, users set this per-share.
-- For workshop shares, the workshop's ai_quality_tier is used instead (this column stays NULL).
ALTER TABLE game_share ADD COLUMN ai_quality_tier text NULL;
