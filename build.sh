#!/bin/bash

set -e # Exit with nonzero exit code if anything fails

cd "$(dirname "$0")"

echo "Building..."

#pushd client
#npm run build
#popd
docker build -t chatgamelab .
