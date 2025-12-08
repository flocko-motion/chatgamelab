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

echo "Starting dev environment (db + proxy)..."
echo ""
echo "Now start the backend and frontend in separate terminals:"
echo "  Terminal 2: ./run-dev-server.sh    (Go server on :8080)"
echo "  Terminal 3: ./run-dev-client.sh  (React on :3000)"
echo ""
echo "Then open http://localhost in your browser"
echo ""

docker compose -f docker-compose.dev.yml up
