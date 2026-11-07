#!/bin/bash

# Integration test runner - Docker lifecycle is managed by test suites
# Each test suite automatically starts/stops its own Docker environment
#
# Requires: gotestsum (go install gotest.tools/gotestsum@latest)
#
# Usage:
#   scripts/test.sh                    # Clean summary (default)
#   scripts/test.sh --verbose          # Full verbose output
#   scripts/test.sh -v                 # Full verbose output (short)

cd "$(dirname "$0")/.."

GOTESTSUM="go run gotest.tools/gotestsum@latest"

# Parse arguments
VERBOSE=false
for arg in "$@"; do
    case "$arg" in
        --verbose|-v) VERBOSE=true ;;
    esac
done

echo "🧪 Running integration tests..."
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

$GOTESTSUM --format "$FORMAT" --format-hide-empty-pkg -- -v ./...

exit $?
