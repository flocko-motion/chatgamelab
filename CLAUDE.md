### Go version

- When schemaf (its codegen, or a dependency it pulls in) needs a newer Go, upgrade: bump `go` in `go/go.mod` and `go.work` and the image in `Dockerfile.gen` together. Following schemaf's Go version is mandatory.

### Code Generation

- **schemaf**: After modifying `schemaf.toml`, `go/db/migrations/`, `go/db/queries/` or the endpoint structs in `go/api/*.go`, regenerate with `./schemaf.sh codegen`
- **OpenAPI → TypeScript** (v1 routes): After modifying Go API route handlers in `go/api/routes` (swagger annotations), regenerate with `./go/generate-openapi.sh` then `cd frontend && npm run gen:api`
