#!/bin/bash
cd "$(dirname "$0")"

if [ ! -f .env ]; then
    echo "Error: .env file not found"
    echo "Copy .env.example to .env and set your values"
    exit 1
fi

source .env

if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD is not set in .env"
    exit 1
fi

echo "Starting production environment..."
docker compose up --build -d
echo ""
echo "Services running at http://localhost"
echo "Use 'docker compose logs -f' to view logs"
echo "Use 'docker compose down' to stop"
