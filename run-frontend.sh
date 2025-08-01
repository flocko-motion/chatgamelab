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