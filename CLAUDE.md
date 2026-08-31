### Code Generation

- **sqlc** (Go DB queries): After modifying `server/db/queries/*.sql` or `server/db/schema.sql`, regenerate with `./server/sqlc.sh`
- **OpenAPI → TypeScript**: After modifying Go API route handlers (swagger annotations), regenerate with `./server/generate-openapi.sh` then `cd web && npm run gen:api`
