# Database Backup Configuration

This database image includes automated backup functionality via SSH.

## Features

- PostgreSQL 18 base image
- SSH client for remote backups
- Gzip compression
- Automated backup script
- A `.sha256` checksum uploaded beside each dump
- Host keys pinned on first sight, on a volume that survives a restart
- A logbook entry per run, reported as backup health on `/api/status`

## Configuration

Set these environment variables in your `.env` file:

```bash
# Enable backups
BACKUP_ENABLED=true

# SSH connection details
BACKUP_SSH_HOST=your-backup-server.com
BACKUP_SSH_PORT=22
BACKUP_SSH_USER=backup-user
BACKUP_SSH_KEY_PATH=/path/to/ssh/private/key
BACKUP_PATH=backups

# Hours before a successful backup is reported as stale (default 48)
BACKUP_MAX_AGE_HOURS=48
```

`BACKUP_ENABLED` and `BACKUP_MAX_AGE_HOURS` are also passed to the backend container,
which needs them to report backup health.

## SSH Key Setup

1. Generate an SSH key pair (if you don't have one):
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/backup_key -N ""
   ```

2. Add the public key to your backup server's `~/.ssh/authorized_keys`

3. Set the path in `.env`:
   ```bash
   BACKUP_SSH_KEY_PATH=/root/.ssh/backup_key
   ```

## Manual Backup

Run a backup manually:

```bash
docker exec chatgamelab-db /usr/local/bin/backup.sh
```

## Automated Backups

For scheduled backups, add a cron job on the host:

```bash
# Daily backup at 2 AM
0 2 * * * docker exec chatgamelab-db /usr/local/bin/backup.sh
```

Or use a systemd timer.

The image ships no scheduler of its own, so without this cron entry backups never run.
That case shows up as `"backup": "unknown"` on `/api/status`.

## Monitoring

Each run appends a row to the `backup_log` table, whether it succeeded or failed:

```bash
docker exec chatgamelab-db psql -U chatgamelab chatgamelab \
  -c "SELECT finished_at, success, filename, size_bytes, error FROM backup_log ORDER BY finished_at DESC LIMIT 10;"
```

The server reads the newest row and reports it on the public `/api/status` endpoint
alongside its uptime:

```json
{ "status": "running", "uptime": "3h 12m", "backup": "ok", "backupAge": "9h 4m" }
```

| `backup`   | Meaning                                                             |
| ---------- | ------------------------------------------------------------------- |
| `disabled` | `BACKUP_ENABLED` is not `true`                                       |
| `ok`       | Newest run succeeded within `BACKUP_MAX_AGE_HOURS`                   |
| `stale`    | Newest run succeeded but is older than that — the schedule has drifted or stopped |
| `failed`   | Newest run failed; `backup_log.error` names the step                 |
| `unknown`  | Enabled, but no run has ever been recorded                           |

Writing the logbook is best-effort: a backup that uploads successfully but cannot
reach the `backup_log` table still counts as a successful backup, and warns on stderr.

## Backup Format

Backups are stored as:
```
{DB_NAME}-{TIMESTAMP}.sql.gz
```

Example: `chatgamelab-2026-02-04_22-30-00.sql.gz`

## Restore

To restore a backup:

```bash
# Download from backup server
scp user@backup-server:backups/chatgamelab-2026-02-04_22-30-00.sql.gz .

# Restore to database
gunzip < chatgamelab-2026-02-04_22-30-00.sql.gz | \
  docker exec -i chatgamelab-db psql -U chatgamelab chatgamelab
```
