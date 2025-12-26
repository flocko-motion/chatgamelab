$ErrorActionPreference = 'Stop'

# Generates OpenAPI spec via swaggo (no swagger UI is served).
# Output:
# - ./docs/swagger.json
# - ./docs/swagger.yaml

$root = Split-Path -Parent $MyInvocation.MyCommand.Path

# Ensure output folder exists
$docsDir = Join-Path $root 'docs'
if (!(Test-Path $docsDir)) {
  New-Item -ItemType Directory -Path $docsDir | Out-Null
}

# Use local toolchain (ensures version pinned in go.mod)
Write-Host 'Running swag init...'

# We point swag at the routes package doc.go as the main entry.
# ParseDependency is needed because annotations reference types in cgl/obj.
# ParseInternal is useful because packages are internal-ish to the module.
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init `
  --generalInfo api/routes/doc.go `
  --dir . `
  --output ./docs `
  --parseDependency `
  --parseInternal

Write-Host 'OpenAPI generated in ./docs'
