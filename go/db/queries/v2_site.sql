-- v2: the site's settings and the backup log, read and written by the
-- secure core (core/site.go).

-- Carried over from v1 unchanged (#344); the core uses these.

-- name: V2GetLatestBackup :one
SELECT
  id,
  started_at,
  finished_at,
  success,
  filename,
  size_bytes,
  error
FROM backup_log
ORDER BY finished_at DESC
LIMIT 1;

-- name: V2GetSystemSettings :one
SELECT
  id,
  created_at,
  modified_at,
  default_ai_quality_tier,
  free_use_ai_quality_tier,
  prompt_constraint_u13,
  prompt_constraint_u13p,
  prompt_constraint_u18,
  free_use_api_key_id
FROM system_settings
LIMIT 1;

-- name: V2UpdateDefaultAiQualityTier :exec
UPDATE system_settings
SET
  default_ai_quality_tier = $1,
  modified_at = now();

-- name: V2UpdateSystemSettingsFreeUseApiKey :exec
UPDATE system_settings
SET
  free_use_api_key_id = $1,
  modified_at = now();

-- name: V2ClearFreeUseKeyOf :exec
-- Clears the site's free-use key if it is one of @user_id's: an admin's own
-- key, gone with their admin role or their account.
UPDATE system_settings SET free_use_api_key_id = NULL, modified_at = now()
WHERE free_use_api_key_id IN (SELECT id FROM api_key WHERE user_id = @user_id);

-- Carried over from v1 for the core's system jobs and settings (#344).

-- name: V2UpdateFreeUseAiQualityTier :exec
UPDATE system_settings
SET
  free_use_ai_quality_tier = $1,
  modified_at = now();

-- name: V2UpdatePromptConstraintU13 :exec
UPDATE system_settings
SET
  prompt_constraint_u13 = $1,
  modified_at = now();

-- name: V2UpdatePromptConstraintU13p :exec
UPDATE system_settings
SET
  prompt_constraint_u13p = $1,
  modified_at = now();

-- name: V2UpdatePromptConstraintU18 :exec
UPDATE system_settings
SET
  prompt_constraint_u18 = $1,
  modified_at = now();
