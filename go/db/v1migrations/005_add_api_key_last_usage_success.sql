-- Add last_usage_success column to api_key table
-- null=unknown, true=last usage succeeded, false=last usage failed
ALTER TABLE api_key ADD COLUMN IF NOT EXISTS last_usage_success boolean NULL;
