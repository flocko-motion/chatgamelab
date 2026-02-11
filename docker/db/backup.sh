#!/bin/bash
set -euo pipefail

# ENV variables (configured via docker-compose)
: "${BACKUP_ENABLED:=false}"
: "${BACKUP_SSH_HOST:?}"
: "${BACKUP_SSH_PORT:=22}"
: "${BACKUP_SSH_USER:?}"
: "${BACKUP_SSH_KEY:=/root/.ssh/backup_key}"
: "${BACKUP_PATH:=backups}"
: "${DB_NAME:?}"
: "${DB_USER:?}"

if [ "$BACKUP_ENABLED" != "true" ]; then
  echo "Backup disabled (BACKUP_ENABLED != true)"
  exit 0
fi

TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
FILENAME="${DB_NAME}-${TIMESTAMP}.sql.gz"
TEMP_FILE="/tmp/${FILENAME}"

echo "Starting backup: $FILENAME"

# Ensure remote directory exists (using SFTP batch mode)
if [ -n "$BACKUP_PATH" ]; then
  echo "Ensuring remote directory exists: $BACKUP_PATH"
  sftp -o StrictHostKeyChecking=no \
    -i "$BACKUP_SSH_KEY" \
    -P "$BACKUP_SSH_PORT" \
    "$BACKUP_SSH_USER@$BACKUP_SSH_HOST" << EOF
-mkdir $BACKUP_PATH
quit
EOF
fi

# Create compressed backup to temp file
pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$TEMP_FILE"

# Upload via scp (works with SFTP-only servers like Hetzner Storage Box)
scp -o StrictHostKeyChecking=no \
  -i "$BACKUP_SSH_KEY" \
  -P "$BACKUP_SSH_PORT" \
  "$TEMP_FILE" \
  "$BACKUP_SSH_USER@$BACKUP_SSH_HOST:$BACKUP_PATH/$FILENAME"

# Cleanup
rm -f "$TEMP_FILE"

echo "Backup complete: $FILENAME"
