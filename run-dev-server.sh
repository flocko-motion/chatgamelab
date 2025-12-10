#!/bin/bash
cd "$(dirname "$0")"
source .env 2>/dev/null || true
cd server
export DATABASE_URL='postgres://chatgamelab:'"${DB_PASSWORD}"'@localhost:5433/chatgamelab?sslmode=disable'
export API_PORT=8080
go run .
