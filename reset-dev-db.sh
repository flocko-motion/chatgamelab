#!/bin/bash
cd "$(dirname "$0")"

echo "This will DELETE all dev database data and recreate from schema.sql"
read -p "Are you sure? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

# Remove all containers and volumes for this project
docker compose -f docker-compose.dev.yml down -v --remove-orphans
