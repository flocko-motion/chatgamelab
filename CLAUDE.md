### Code Generation

- **schemaf**: After modifying `schemaf.toml`, `go/db/migrations/`, `go/db/queries/` or the endpoint structs in `go/api/*.go`, regenerate with `./schemaf.sh codegen`
- **sqlc** (v1 queries): After modifying `go/db/sqlc/queries/*.sql`, regenerate with `./go/sqlc.sh`
- **OpenAPI → TypeScript** (v1 routes): After modifying Go API route handlers in `go/api/routes` (swagger annotations), regenerate with `./go/generate-openapi.sh` then `cd frontend && npm run gen:api`
