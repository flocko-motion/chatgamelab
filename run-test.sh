#!/bin/bash

# Simple test runner - Docker lifecycle is managed by test suites
# Each test suite automatically starts/stops its own Docker environment

cd "$(dirname "$0")"

set -e

echo "🧪 Running integration tests..."
echo ""

cd testing
go test -v ./...
TEST_EXIT_CODE=$?
cd ..

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Tests failed with exit code $TEST_EXIT_CODE"
fi

exit $TEST_EXIT_CODE
