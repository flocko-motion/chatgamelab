#!/bin/bash
set -euo pipefail

# ENV variables (configured via docker-compose)
: "${BACKUP_ENABLED:=false}"

# Checked before anything is required, so a deployment with backups switched
# off exits cleanly instead of failing on an unset variable it does not need.
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

# Host keys are remembered across runs rather than ignored. StrictHostKeyChecking=no
# accepts whatever answers, every time, and what leaves here is a dump of the whole
# database — including users' provider API keys. accept-new pins on first sight and
# refuses a changed key after that. The file lives on the data volume so the pin
# survives a container restart.
KNOWN_HOSTS=/var/lib/postgresql/.ssh/known_hosts
mkdir -p "$(dirname "$KNOWN_HOSTS")"
touch "$KNOWN_HOSTS"

# BatchMode=yes is what makes a failure a failure. Without it, ssh answers a
# refused key by prompting for a password, reads the heredoc below as the
# answer, and hangs until something kills it — which is a backup that reports
# a timeout every night while nothing is wrong with the database at all.
SFTP_OPTS=(
  -o BatchMode=yes
  -o StrictHostKeyChecking=accept-new
  -o UserKnownHostsFile="$KNOWN_HOSTS"
  -o ConnectTimeout=30
  -i "$BACKUP_SSH_KEY"
  -P "$BACKUP_SSH_PORT"
)
REMOTE="$BACKUP_SSH_USER@$BACKUP_SSH_HOST"

STARTED_AT=$(date -Iseconds)
TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
FILENAME="${DB_NAME}-${TIMESTAMP}.sql.gz"
TEMP_FILE="/tmp/${FILENAME}"
STEP="startup"
trap 'rm -f "$TEMP_FILE" "$TEMP_FILE.sha256"' EXIT

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

# A command on the left of || is invisible to the ERR trap, so the ones that
# diagnose themselves report through here and every way out still leaves an entry.
fail() { # message
  log_entry false "$FILENAME" "" "$1"
  echo "Backup failed: $1" >&2
  exit 1
}

on_error() {
  local code=$?
  log_entry false "$FILENAME" "" "$STEP failed (exit $code)"
  echo "Backup failed during $STEP (exit $code)" >&2
  exit "$code"
}
trap on_error ERR

echo "Starting backup: $FILENAME"

# One component at a time: sftp's mkdir makes a single level, so a nested
# BACKUP_PATH silently never appeared and every upload afterwards had nowhere
# to go. Each mkdir is prefixed with - because the directory usually exists;
# whether it worked is settled by the cd below, which is not.
if [ -n "$BACKUP_PATH" ]; then
  echo "Ensuring remote directory exists: $BACKUP_PATH"
  STEP="remote mkdir"
  {
    path=""
    IFS=/ read -ra parts <<<"$BACKUP_PATH"
    for part in "${parts[@]}"; do
      [ -n "$part" ] || continue
      path="${path:+$path/}$part"
      echo "-mkdir $path"
    done
    echo "cd $BACKUP_PATH"
    echo "quit"
  } | sftp "${SFTP_OPTS[@]}" -b - "$REMOTE" \
    || fail "cannot reach $REMOTE:$BACKUP_PATH — check the key, the account and the path"
fi

# Compressed backup to a temp file
STEP="pg_dump"
pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$TEMP_FILE"
sha256sum "$TEMP_FILE" | cut -d' ' -f1 > "$TEMP_FILE.sha256"
SIZE=$(stat -c %s "$TEMP_FILE")
echo "Dumped $(du -h "$TEMP_FILE" | cut -f1)"

# The checksum goes up beside the dump, so a truncated upload can be told from
# a good one without downloading and decompressing it.
STEP="sftp upload"
sftp "${SFTP_OPTS[@]}" -b - "$REMOTE" <<UPLOAD
put "$TEMP_FILE" "$BACKUP_PATH/$FILENAME"
put "$TEMP_FILE.sha256" "$BACKUP_PATH/$FILENAME.sha256"
quit
UPLOAD

log_entry true "$FILENAME" "$SIZE" ""

echo "Backup complete: $FILENAME ($SIZE bytes)"
