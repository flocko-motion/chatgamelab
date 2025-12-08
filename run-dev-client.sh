#!/bin/bash

set -e # Exit with nonzero exit code if anything fails

cd "$(dirname "$0")"

echo "Starting Chat Game Lab Frontend..."
echo ""

# Check if we're in the right directory
if [ ! -d "client" ]; then
    echo "❌ Error: client directory not found"
    echo "   Make sure you're running this from the project root"
    exit 1
fi

# Check if Node.js and npm are installed
if ! command -v node >/dev/null 2>&1; then
    echo "❌ Error: Node.js is not installed"
    echo "   Please install Node.js and npm from: https://nodejs.org/"
    echo "   This will install both Node.js and npm which are required to run the frontend."
    exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
    echo "❌ Error: npm is not installed"
    echo "   Please install Node.js and npm from: https://nodejs.org/"
    echo "   This will install both Node.js and npm which are required to run the frontend."
    exit 1
fi

# Generate version file with git info
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "export const version = \"${GIT_COMMIT}\";" > client/src/version.js
echo "export const buildTime = \"${BUILD_TIME}\";" >> client/src/version.js

# Move to client directory
cd client

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
    echo ""
fi

echo "🚀 Starting React development server..."
echo ""
echo "Frontend will be available at:"
echo "  Regular mode: http://localhost:3000"
echo "  Mock mode:    http://localhost:3000?mock=true"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Start the React dev server
npm start