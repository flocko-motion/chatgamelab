#!/bin/bash
cd "$(dirname "$0")"

if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD is not set"
    exit 1
fi

echo "Starting production environment..."
docker compose up --build -d
echo ""
echo "Services running at:"
echo "http://127.0.0.1:${PORT_EXPOSED}"
echo "${PUBLIC_URL}"
echo "Use 'docker compose logs -f' to view logs"
echo "Use 'docker compose down' to stop"
