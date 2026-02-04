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

echo "Starting backup: $FILENAME"

pg_dump -U "$DB_USER" "$DB_NAME" | \
  gzip | \
  ssh -o StrictHostKeyChecking=no \
    -i "$BACKUP_SSH_KEY" \
    -p "$BACKUP_SSH_PORT" \
    "$BACKUP_SSH_USER@$BACKUP_SSH_HOST" \
    "cat > $BACKUP_PATH/$FILENAME"

echo "Backup complete: $FILENAME"
