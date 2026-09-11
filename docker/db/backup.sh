#!/bin/bash
set -euo pipefail

# ENV variables (configured via docker-compose)
: "${BACKUP_ENABLED:=false}"

if [ "$BACKUP_ENABLED" != "true" ]; then
  echo "Backup disabled (BACKUP_ENABLED != true)"
  exit 0
fi

: "${BACKUP_SSH_HOST:?}"
: "${BACKUP_SSH_PORT:=22}"
: "${BACKUP_SSH_USER:?}"
: "${BACKUP_SSH_KEY:=/root/.ssh/backup_key}"
: "${BACKUP_PATH:=backups}"
: "${DB_NAME:?}"
: "${DB_USER:?}"

STARTED_AT=$(date -Iseconds)
TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
FILENAME="${DB_NAME}-${TIMESTAMP}.sql.gz"
TEMP_FILE="/tmp/${FILENAME}"
STEP="startup"

# Append one entry to the backup_log table read by the server's /api/status endpoint.
# Best-effort: a logbook that cannot be written must never turn a good backup into a
# failed one, so a psql error only warns.
# The SQL arrives on stdin because psql interpolates :'vars' only when reading a
# script, never in a -c string. Interpolation quotes each value, so an error message
# containing quotes is stored verbatim instead of breaking the statement.
log_entry() { # success filename size error
  if ! psql -q -U "$DB_USER" -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -v started="$STARTED_AT" -v success="$1" -v filename="$2" -v size="$3" -v err="$4" \
    > /dev/null <<'SQL'
INSERT INTO backup_log (started_at, finished_at, success, filename, size_bytes, error)
VALUES (:'started'::timestamptz, now(), :'success'::boolean,
        NULLIF(:'filename', ''), NULLIF(:'size', '')::bigint, NULLIF(:'err', ''));
SQL
  then
    echo "Warning: could not write backup_log entry" >&2
  fi
}

on_error() {
  local code=$?
  log_entry false "$FILENAME" "" "$STEP failed (exit $code)"
  rm -f "$TEMP_FILE"
  echo "Backup failed during $STEP (exit $code)" >&2
  exit "$code"
}
trap on_error ERR

echo "Starting backup: $FILENAME"

# Ensure remote directory exists (using SFTP batch mode)
if [ -n "$BACKUP_PATH" ]; then
  echo "Ensuring remote directory exists: $BACKUP_PATH"
  STEP="remote mkdir"
  sftp -o StrictHostKeyChecking=no \
    -i "$BACKUP_SSH_KEY" \
    -P "$BACKUP_SSH_PORT" \
    "$BACKUP_SSH_USER@$BACKUP_SSH_HOST" << EOF
-mkdir $BACKUP_PATH
quit
EOF
fi

# Create compressed backup to temp file
STEP="pg_dump"
pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$TEMP_FILE"
SIZE=$(stat -c %s "$TEMP_FILE")

# Upload via SFTP (works with SFTP-only servers like Hetzner Storage Box)
STEP="sftp upload"
sftp -o StrictHostKeyChecking=no \
  -i "$BACKUP_SSH_KEY" \
  -P "$BACKUP_SSH_PORT" \
  "$BACKUP_SSH_USER@$BACKUP_SSH_HOST" <<UPLOAD
put "$TEMP_FILE" "$BACKUP_PATH/$FILENAME"
quit
UPLOAD

# Cleanup
rm -f "$TEMP_FILE"

log_entry true "$FILENAME" "$SIZE" ""

echo "Backup complete: $FILENAME ($SIZE bytes)"
