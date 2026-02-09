-- Rename allow_public_sponsored_plays -> allow_public_game_sponsoring on api_key_share
ALTER TABLE api_key_share RENAME COLUMN allow_public_sponsored_plays TO allow_public_game_sponsoring;

-- Add free_use_api_key_share_id to institution (nullable FK to api_key_share)
ALTER TABLE institution ADD COLUMN IF NOT EXISTS free_use_api_key_share_id uuid NULL REFERENCES api_key_share(id);

-- Add free_use_api_key_id to system_settings (nullable FK to api_key, admin-only)
ALTER TABLE system_settings ADD COLUMN IF NOT EXISTS free_use_api_key_id uuid NULL REFERENCES api_key(id);
