#!/bin/bash
cd "$(dirname "$0")"

echo "This will DELETE all dev database data and recreate from schema.sql"
read -p "Are you sure? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

docker compose -f docker-compose.dev.yml down -v
echo "Database volume removed. Run ./run-dev.sh to recreate with fresh schema."
