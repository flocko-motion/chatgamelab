#!/bin/bash
# Push current branch with automatic rebase if remote has diverged.
# Usage: ./git-push.sh

set -e
cd "$(dirname "$0")"

BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "Pushing $BRANCH..."

if git push origin "$BRANCH" 2>/dev/null; then
    echo "✅ Pushed successfully."
    exit 0
fi

echo "⚠️  Push rejected - rebasing on origin/development..."
git fetch origin development
git rebase origin/development

echo "Pushing again after rebase..."
git push origin "$BRANCH"
echo "✅ Pushed successfully after rebase."
