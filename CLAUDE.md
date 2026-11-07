### Code Generation

- **sqlc** (Go DB queries): After modifying `platform/server/db/queries/*.sql` or `platform/server/db/schema.sql`, regenerate with `./platform/server/sqlc.sh`
- **OpenAPI → TypeScript**: After modifying Go API route handlers (swagger annotations), regenerate with `./platform/server/generate-openapi.sh` then `cd platform/web && npm run gen:api`
