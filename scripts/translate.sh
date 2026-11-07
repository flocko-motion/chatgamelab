#!/bin/bash
# Translates the platform's locale files from en + de into every other language.
# With --check it calls no AI and fails if anything needs translating.
#
# Usage: scripts/translate.sh [--check] [platform] [threads]

set -e

CHECK=""
if [ "$1" = "--check" ]; then
    CHECK=1
    shift
fi
PLATFORM="${1:-openai}"
THREADS="${2:-0}"

cd "$(dirname "$0")/../platform/server"
LOCALES=../web/src/i18n/locales

if [ -n "$CHECK" ]; then
    go run . lang translate --check --input "$LOCALES" --output "$LOCALES"
    if ! go run . lang list --json | cmp -s - "$LOCALES/languages.json"; then
        echo "✗ languages.json is stale — run make translate" >&2
        exit 1
    fi
    for f in "$LOCALES"/*.json; do
        if ! cmp -s "$f" "lang/locales/$(basename "$f")"; then
            echo "✗ platform/server/lang/locales is out of sync — run make translate" >&2
            exit 1
        fi
    done
    echo "✓ Translations are complete"
    exit 0
fi

set -x
time go run . lang translate --platform "$PLATFORM" --threads "$THREADS" --input "$LOCALES" --output "$LOCALES"
go run . lang list --json > "$LOCALES/languages.json"
rm ./lang/locales/*.json
cp "$LOCALES"/*.json ./lang/locales/.
