-- Add is_default column to api_key table
ALTER TABLE api_key ADD COLUMN IF NOT EXISTS is_default boolean NOT NULL DEFAULT false;

-- Create unique partial index: only one default key per user
CREATE UNIQUE INDEX IF NOT EXISTS api_key_user_default_uniq ON api_key (user_id) WHERE is_default = true;

-- Migrate existing defaults: for each user that has a default_api_key_share_id,
-- set the corresponding api_key.is_default = true
UPDATE api_key k
SET is_default = true
FROM app_user u
JOIN api_key_share s ON s.id = u.default_api_key_share_id
WHERE k.id = s.api_key_id
  AND k.user_id = u.id
  AND u.default_api_key_share_id IS NOT NULL;

-- For users who have api keys but no default set, make their first key the default
UPDATE api_key
SET is_default = true
WHERE id IN (
  SELECT DISTINCT ON (user_id) id
  FROM api_key
  WHERE user_id NOT IN (
    SELECT user_id FROM api_key WHERE is_default = true
  )
  ORDER BY user_id, created_at ASC
);
