#!/usr/bin/env bash
# Opens and closes the agents' seal in one issue worktree, as the owner.
# See docs/workflows/opening-the-seal.md.
#
#   scripts/seal.sh open     # agents may edit the sealed paths here
#   scripts/seal.sh close    # back to the committed seal
#   scripts/seal.sh status
#
# open drops the Edit and Write deny rules from this worktree's
# .claude/settings.json; the Bash denials (admin merges, rulesets) stay. It
# marks the file skip-worktree in this worktree's index, so git sees no change
# to it and no `git add` or `git commit -a` picks the open seal up. close
# lifts the mark and restores the committed file.
#
# The main checkout stays sealed, and an agent may close the seal but never
# open it.

set -euo pipefail

SETTINGS=.claude/settings.json

die() {
    echo "seal: $*" >&2
    exit 1
}

cd "$(git rev-parse --show-toplevel)"

skipped() {
    [[ "$(git ls-files -v -- "$SETTINGS")" == S* ]]
}

is_open() {
    ! git diff --quiet HEAD -- "$SETTINGS" 2>/dev/null || skipped
}

open_seal() {
    [[ -z "${CLAUDECODE:-}" ]] || die "an agent can't open the seal; the owner runs this"
    [[ "$(git rev-parse --absolute-git-dir)" != "$(git rev-parse --path-format=absolute --git-common-dir)" ]] ||
        die "the main checkout stays sealed; open the seal in the issue's worktree"
    command -v jq >/dev/null || die "needs jq"
    git diff --cached --quiet -- "$SETTINGS" || die "$SETTINGS has staged changes; unstage them first"
    if is_open; then
        echo "seal: already open"
        return
    fi

    local tmp
    tmp="$(mktemp)"
    jq '.permissions.deny |= map(select(test("^(Edit|Write)\\(") | not))' "$SETTINGS" >"$tmp"
    mv "$tmp" "$SETTINGS"
    git update-index --skip-worktree -- "$SETTINGS"
    echo "seal: open in $(pwd); git ignores the change to $SETTINGS here"
}

close_seal() {
    if skipped; then
        git update-index --no-skip-worktree -- "$SETTINGS"
    fi
    git checkout HEAD -- "$SETTINGS"
    echo "seal: closed"
}

case "${1:-}" in
open) open_seal ;;
close) close_seal ;;
status) if is_open; then echo "open"; else echo "sealed"; fi ;;
*) die "usage: scripts/seal.sh open|close|status" ;;
esac
