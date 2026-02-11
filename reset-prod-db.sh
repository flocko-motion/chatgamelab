#!/bin/bash

# Reset production database on remote webserver
# Usage: 
#   ./reset-prod-db.sh [user@host] [remote_dir]
#   ./reset-prod-db.sh webmaster@omnitopos.net /opt/chatgamelab
#
# If arguments are not provided, the script will prompt for them.

# Get remote host from argument or prompt
if [ -n "$1" ]; then
    REMOTE_HOST="$1"
else
    read -p "Enter remote host (user@hostname): " REMOTE_HOST
    if [ -z "$REMOTE_HOST" ]; then
        echo "Error: Remote host is required"
        exit 1
    fi
fi

# Get remote directory from argument or prompt
if [ -n "$2" ]; then
    REMOTE_DIR="$2"
else
    read -p "Enter remote directory [/opt/chatgamelab]: " REMOTE_DIR
    REMOTE_DIR="${REMOTE_DIR:-/opt/chatgamelab}"
fi

echo ""
echo "⚠️  WARNING: This will DELETE all database data on $REMOTE_HOST!"
echo "Remote directory: $REMOTE_DIR"
echo "The database will be reinitialized from the embedded schema on next startup."
echo ""
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

echo "Connecting to $REMOTE_HOST..."

ssh "$REMOTE_HOST" << ENDSSH
cd $REMOTE_DIR

echo "Stopping containers..."
docker compose down --remove-orphans

echo "Removing database container..."
docker rm -f chatgamelab-db 2>/dev/null || true

echo "Removing database volume..."
docker volume rm chatgamelab_db_data 2>/dev/null || true

echo "✅ Database volumes removed."
echo ""
echo "Starting services..."
docker compose up -d

echo ""
echo "✅ Services restarted. Database will be automatically initialized."
ENDSSH

echo ""
echo "✅ Remote database reset complete!"
echo "Services are now running on $REMOTE_HOST"
