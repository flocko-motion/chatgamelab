#!/bin/bash

set -e # Exit with nonzero exit code if anything fails

cd "$(dirname "$0")"

echo "Starting up..."

docker run -p 3000:3000 -v ./server/var:/app/var chatgamelab
