---
trigger: model_decision
description: Apply when working with backend server and cgl CLI tool
---
We use sqlc for database queries. The queries live in `go/db/queries/`; `./schemaf.sh codegen` generates them into package `db`.
Backend is at /server. 
OpenAPI docs are at /go/api/docs and can be generated with /go/generate-openapi.sh
All db access is regulated by /go/db/permissions.go - don't implement permission logic in other places.
Strong conventions are used all through the backend - thus always read the existing code to understand the logic and follow the convention.