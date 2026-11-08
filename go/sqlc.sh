#!/usr/bin/env bash
set -euo pipefail

# Run sqlc code generation for the server.
# Usage (from anywhere):
#   ./server/sqlc.sh

# Change to the directory where this script lives, so relative paths work.
cd "$(dirname "${BASH_SOURCE[0]}")"
echo "running in $(pwd)"

go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

SQLC_BIN="$(go env GOPATH)/bin/sqlc"
"$SQLC_BIN" generate -f "$(pwd)/sqlc.yaml"
