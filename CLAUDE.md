## MANDATORY: Use td for Task Management

Run `td usage --new-session` at conversation start (or after /clear). This tells you what to work on next.

Sessions are automatic (based on terminal/agent context). Optional:
- `td session "name"` to label the current session
- `td session --new` to force a new session in the same context

### td workflow

1. `td next` — get next prioritized issue
2. `td start <id>` — begin working on it
3. `td handoff <id> --done "..." --decision "..." --remaining "..."` — capture working state when done
4. `td review <id>` — submit for review (always do this immediately after handoff)

When user says "next", run `td next` (not `td list`).
Always complete handoff → review in sequence without waiting to be prompted.

Use `td usage -q` after first read for quick reference.

### Code Generation

- **sqlc** (Go DB queries): After modifying `server/db/queries/*.sql` or `server/db/schema.sql`, regenerate with `./server/sqlc.sh`
- **OpenAPI → TypeScript**: After modifying Go API route handlers (swagger annotations), regenerate with `./server/generate-openapi.sh` then `cd web && npm run gen:api`
