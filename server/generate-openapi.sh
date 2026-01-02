#!/usr/bin/env bash
set -euo pipefail

# Generates OpenAPI spec via swaggo (no swagger UI is served).
# Output:
# - ./docs/swagger.json
# - ./docs/swagger.yaml

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCS_DIR="$ROOT_DIR/docs"

mkdir -p "$DOCS_DIR"

echo "Running swag init..."

# We point swag at the routes package doc.go as the main entry.
# parseDependency is needed because annotations reference types in cgl/obj.
# parseInternal is useful because packages are internal-ish to the module.
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init \
  --generalInfo api/routes/doc.go \
  --dir . \
  --output ./docs \
  --parseDependency \
  --parseInternal

echo "OpenAPI generated in ./docs"
