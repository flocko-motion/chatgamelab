-- Backup Log queries

-- name: GetLatestBackup :one
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
