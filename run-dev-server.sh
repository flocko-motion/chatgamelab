#!/bin/bash
cd "$(dirname "$0")"
source .env 2>/dev/null || true
cd server
# using the --dev arg allows logging into any user account bypassing auth0
# (useful for development without needing Auth0 credentials)
go run . server --dev
