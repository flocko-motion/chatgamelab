#!/bin/bash

# Integration test runner - Docker lifecycle is managed by test suites
# Each test suite automatically starts/stops its own Docker environment
#
# Requires: gotestsum (go install gotest.tools/gotestsum@latest)
#
# Usage:
#   ./run-test.sh                    # Clean summary (default)
#   ./run-test.sh --verbose          # Full verbose output
#   ./run-test.sh -v                 # Full verbose output (short)
#   ./run-test.sh --no-ai            # Exclude AI tests (default)
#   ./run-test.sh --ai               # Include AI tests
#   ./run-test.sh --only-ai          # Run ONLY AI tests
#   ./run-test.sh --only-ai --verbose # Run only AI tests, verbose

cd "$(dirname "$0")"

GOTESTSUM="go run gotest.tools/gotestsum@latest"

# Parse arguments
BUILD_TAGS=""
TEST_RUN=""
VERBOSE=false
for arg in "$@"; do
    case "$arg" in
        --no-ai)   ;;  # default behavior, no-op
        --verbose|-v) VERBOSE=true ;;
        --ai)      BUILD_TAGS="-tags ai_tests" ;;
        --only-ai) BUILD_TAGS="-tags ai_tests" ; TEST_RUN="-run TestGameEngineSuite" ;;
    esac
done

if [[ "$TEST_RUN" == "-run TestGameEngineSuite" ]]; then
    echo "🧪 Running ONLY AI tests..."
elif [[ -n "$BUILD_TAGS" ]]; then
    echo "🧪 Running integration tests (including AI tests)..."
else
    echo "🧪 Running integration tests (excluding AI tests)..."
fi
echo ""

cd testing

# Pick format: CI gets collapsible groups, terminal gets compact output
if [[ -n "$CI" ]]; then
    FORMAT="github-actions"
elif $VERBOSE; then
    FORMAT="standard-verbose"
else
    # Show dots for progress, then detailed failure summary at the end
    FORMAT="dots-v2"
fi

$GOTESTSUM --format "$FORMAT" --format-hide-empty-pkg -- -v $BUILD_TAGS $TEST_RUN ./...

exit $?
