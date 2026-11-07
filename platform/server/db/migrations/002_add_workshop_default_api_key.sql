-- Add default API key share column to workshop table
-- This allows staff/heads to set a default API key for workshop participants

ALTER TABLE workshop ADD COLUMN default_api_key_share_id uuid NULL REFERENCES api_key_share(id);
