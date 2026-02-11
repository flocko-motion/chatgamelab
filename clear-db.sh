#!/bin/bash
# clear-db.sh - Clear all data from chatgamelab database schema

set -e

CONTAINER="chatgamelab-db"
DB_USER="chatgamelab"
DB_NAME="chatgamelab"

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
    echo "Error: Container ${CONTAINER} is not running"
    exit 1
fi

echo "⚠️  WARNING: This will delete ALL data in the database '${DB_NAME}'"
echo "This action cannot be undone!"
echo

# Ask for confirmation
read -p "Are you sure you want to continue? (type 'yes' to confirm): " confirmation
if [ "$confirmation" != "yes" ]; then
    echo "Operation cancelled."
    exit 0
fi

echo "Clearing database schema in container ${CONTAINER}..."

# Drop and recreate the entire database schema
docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" << 'EOF'
-- Disable foreign key checks temporarily
SET session_replication_role = replica;

-- Drop all tables in the correct order (child tables first)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename) LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
        RAISE NOTICE 'Dropped table: %', r.tablename;
    END LOOP;
END $$;

-- Drop all sequences
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public') LOOP
        EXECUTE 'DROP SEQUENCE IF EXISTS ' || quote_ident(r.sequence_name) || ' CASCADE';
        RAISE NOTICE 'Dropped sequence: %', r.sequence_name;
    END LOOP;
END $$;

-- Drop all types
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT typname FROM pg_type WHERE typtype = 'e' AND typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')) LOOP
        EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
        RAISE NOTICE 'Dropped type: %', r.typname;
    END LOOP;
END $$;

-- Drop all functions
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT proname FROM pg_proc WHERE pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')) LOOP
        EXECUTE 'DROP FUNCTION IF EXISTS ' || quote_ident(r.proname) || '() CASCADE';
        RAISE NOTICE 'Dropped function: %', r.proname;
    END LOOP;
END $$;

-- Re-enable foreign key checks
SET session_replication_role = DEFAULT;

-- Vacuum to clean up
VACUUM FULL;
EOF

echo "✅ Database cleared successfully"
echo "All tables, sequences, types, and functions have been removed from '${DB_NAME}'"
