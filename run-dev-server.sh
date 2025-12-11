#!/bin/bash
cd "$(dirname "$0")"
source .env 2>/dev/null || true
cd server
export DATABASE_URL='postgres://chatgamelab@localhost:5433/chatgamelab?sslmode=disable'
go run .
