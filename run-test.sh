#!/bin/bash

# Simple test runner - Docker lifecycle is managed by test suites
# Each test suite automatically starts/stops its own Docker environment
#
# Usage:
#   ./run-test.sh           # Run all tests including AI tests
#   ./run-test.sh --no-ai   # Run tests excluding AI tests (for CI/CD)

cd "$(dirname "$0")"

set -e

# Parse arguments
BUILD_TAGS=""
if [[ "$1" == "--no-ai" ]]; then
    echo "🧪 Running integration tests (excluding AI tests)..."
    BUILD_TAGS=""
else
    echo "🧪 Running integration tests (including AI tests)..."
    BUILD_TAGS="-tags ai_tests"
fi
echo ""

cd testing
go test -v $BUILD_TAGS ./...
TEST_EXIT_CODE=$?
cd ..

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Tests failed with exit code $TEST_EXIT_CODE"
fi

exit $TEST_EXIT_CODE
