-- System Settings queries

-- name: GetSystemSettings :one
SELECT
  id,
  created_at,
  modified_at,
  default_ai_quality_tier,
  free_use_ai_quality_tier,
  free_use_api_key_id
FROM system_settings
LIMIT 1;

-- name: UpdateDefaultAiQualityTier :exec
UPDATE system_settings
SET
  default_ai_quality_tier = $1,
  modified_at = now();

-- name: UpdateFreeUseAiQualityTier :exec
UPDATE system_settings
SET
  free_use_ai_quality_tier = $1,
  modified_at = now();

-- name: UpdateSystemSettingsFreeUseApiKey :exec
UPDATE system_settings
SET
  free_use_api_key_id = $1,
  modified_at = now();

-- name: InitSystemSettings :exec
INSERT INTO system_settings (id, default_ai_quality_tier)
VALUES ('00000000-0000-0000-0000-000000000001'::uuid, $1)
ON CONFLICT DO NOTHING;
