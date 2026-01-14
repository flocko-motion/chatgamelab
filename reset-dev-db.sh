#!/bin/bash
cd "$(dirname "$0")"

echo "This will DELETE all dev database data and recreate from schema.sql"
read -p "Are you sure? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

# Remove all containers and orphans (containers should already be stopped)
docker compose -f docker-compose.dev.yml down --remove-orphans

# Force remove the database container if it still exists
docker rm -f chatgamelab-db 2>/dev/null || true

# Explicitly remove the named volume (down -v doesn't remove named volumes)
docker volume rm chatgamelab_db_data 2>/dev/null || true

echo "✅ Database volumes removed. Fresh database will be initialized on next startup."
