-- Plays through the public page are counted per workshop instead of per game.
-- public_play_limit NULL means unlimited; a leader may cap the total to bound the cost.
ALTER TABLE workshop ADD COLUMN public_play_limit integer NULL;
ALTER TABLE workshop ADD COLUMN public_play_used integer NOT NULL DEFAULT 0;

-- The public page offers joining the workshop through its open invite.
ALTER TABLE workshop ADD COLUMN public_join boolean NOT NULL DEFAULT false;

-- The language new participants start in; existing workshops take their creator's.
ALTER TABLE workshop ADD COLUMN language text NOT NULL DEFAULT 'en';
UPDATE workshop w SET language = u.language FROM app_user u WHERE u.id = w.created_by;

-- The public page's links each allowed 50 plays; the workshop total replaces that.
UPDATE game_share SET remaining = NULL WHERE public_page;
