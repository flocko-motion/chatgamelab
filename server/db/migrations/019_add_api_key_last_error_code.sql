-- Migration 019: Add last_error_code column to api_key
-- Stores the machine-readable error code from the last failed API call.
-- Cleared (set to NULL) when the key is used successfully.
-- Allows showing key owners WHY their key isn't working.

ALTER TABLE api_key ADD COLUMN IF NOT EXISTS last_error_code text NULL;
