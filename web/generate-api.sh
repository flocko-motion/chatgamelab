#!/usr/bin/env bash
set -euo pipefail

# Generates frontend API client from backend OpenAPI spec
# Steps:
# 1. Generate OpenAPI spec in server
# 2. Copy swagger.json to web directory
# 3. Run swagger-typescript-api to generate TypeScript client

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$SCRIPT_DIR/../server"
WEB_DIR="$SCRIPT_DIR"

echo "Step 1: Generating OpenAPI spec in server..."
cd "$SERVER_DIR"
./generate-openapi.sh

echo "Step 2: Copying swagger.json to web directory..."
cp "$SERVER_DIR/docs/swagger.json" "$WEB_DIR/swagger.json"

echo "Step 3: Generating TypeScript API client..."
cd "$WEB_DIR"
npm run gen:api

echo "✅ API client generation complete!"
