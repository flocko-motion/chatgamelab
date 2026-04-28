#!/bin/bash
# Push current branch with automatic rebase if remote has diverged.
# Usage: ./git-push.sh

set -e
cd "$(dirname "$0")"

echo "Rebasing to origin/development..."

git fetch origin development
git rebase origin/development
git push --force-with-lease

echo "✅ Rebased successfully."
