---
trigger: model_decision
description: Apply when working with backend server and cgl CLI tool
---
We use sqlc for database queries. sqlc is configured to generate queries in the `platform/server/db/queries` directory. Run "sqlc.sh" in the "platform/server" directory to generate queries.
Backend is at /server. 
OpenAPI docs are at /platform/server/api/docs and can be generated with /platform/server/generate-openapi.sh
All db access is regulated by /platform/server/db/permissions.go - don't implement permission logic in other places.
Strong conventions are used all through the backend - thus always read the existing code to understand the logic and follow the convention.