# Database Backup Configuration

This database image includes automated backup functionality via SSH.

## Features

- PostgreSQL 18 base image
- SSH client for remote backups
- Gzip compression
- Automated backup script

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
```

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
