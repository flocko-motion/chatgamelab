#!/usr/bin/env bash
# Opens and closes the agents' seal in one issue worktree, as the owner.
# See docs/workflows/opening-the-seal.md.
#
#   scripts/seal.sh open [--for <time>]  # agents may edit the sealed paths here, for a while
#   scripts/seal.sh close                # back to the committed seal
#   scripts/seal.sh status
#
# open drops the Edit and Write deny rules from this worktree's
# .claude/settings.json; the Bash denials (admin merges, rulesets) stay. It
# marks the file skip-worktree in this worktree's index, so git sees no change
# to it and no `git add` or `git commit -a` picks the open seal up. close
# lifts the mark and restores the committed file.
#
# open stays in the foreground and counts down; when the time is up, or on
# Ctrl-C, it closes the seal again. The time is 5 minutes unless --for names
# another: 10m, 90s, 1h, or a bare number of minutes.
#
# The main checkout stays sealed, and an agent may close the seal but never
# open it.

set -euo pipefail

SETTINGS=.claude/settings.json
DEFAULT_FOR=5m

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

# seconds turns 10m, 90s, 1h or a bare number of minutes into seconds.
seconds() {
    local n
    case "$1" in
    *h) n="${1%h}"; [[ "$n" =~ ^[0-9]+$ ]] && echo $((n * 3600)) && return ;;
    *m) n="${1%m}"; [[ "$n" =~ ^[0-9]+$ ]] && echo $((n * 60)) && return ;;
    *s) n="${1%s}"; [[ "$n" =~ ^[0-9]+$ ]] && echo "$n" && return ;;
    *) [[ "$1" =~ ^[0-9]+$ ]] && echo $(($1 * 60)) && return ;;
    esac
    die "--for takes a time like 10m, 90s or 1h, not '$1'"
}

# clock shows seconds as m:ss, or h:mm:ss from an hour on.
clock() {
    local s=$1
    if ((s >= 3600)); then
        printf '%d:%02d:%02d' $((s / 3600)) $((s % 3600 / 60)) $((s % 60))
    else
        printf '%d:%02d' $((s / 60)) $((s % 60))
    fi
}

open_seal() {
    local spec=$DEFAULT_FOR
    while (($#)); do
        case "$1" in
        --for) [[ $# -ge 2 ]] || die "--for needs a time, e.g. --for 10m"; spec=$2; shift 2 ;;
        --for=*) spec=${1#--for=}; shift ;;
        *) die "usage: scripts/seal.sh open [--for <time>]" ;;
        esac
    done
    local total
    total=$(seconds "$spec")
    ((total > 0)) || die "--for must be longer than nothing"

    [[ -z "${CLAUDECODE:-}" ]] || die "an agent can't open the seal; the owner runs this"
    [[ "$(git rev-parse --absolute-git-dir)" != "$(git rev-parse --path-format=absolute --git-common-dir)" ]] ||
        die "the main checkout stays sealed; open the seal in the issue's worktree"
    command -v jq >/dev/null || die "needs jq"
    git diff --cached --quiet -- "$SETTINGS" || die "$SETTINGS has staged changes; unstage them first"
    if is_open; then
        die "already open; scripts/seal.sh close first, then open again"
    fi

    local tmp
    tmp="$(mktemp)"
    jq '.permissions.deny |= map(select(test("^(Edit|Write)\\(") | not))' "$SETTINGS" >"$tmp"
    mv "$tmp" "$SETTINGS"
    git update-index --skip-worktree -- "$SETTINGS"
    trap 'close_on_exit' EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM HUP

    echo "seal: open in $(pwd); git ignores the change to $SETTINGS here"
    if [[ "$spec" == "$DEFAULT_FOR" ]]; then
        echo "seal: closes by itself in $(clock "$total"); for longer: scripts/seal.sh open --for 30m"
    else
        echo "seal: closes by itself in $(clock "$total")"
    fi
    echo "seal: Ctrl-C closes it now"

    local end=$((SECONDS + total)) left
    while :; do
        left=$((end - SECONDS))
        ((left > 0)) || break
        if [[ -t 1 ]]; then
            printf '\rseal: open, closes in %s ' "$(clock "$left")"
        elif ((left % 60 == 0)); then
            echo "seal: open, closes in $(clock "$left")"
        fi
        sleep 1
    done
    [[ -t 1 ]] && printf '\r\033[K'
    echo "seal: time is up"
}

close_on_exit() {
    [[ -t 1 ]] && printf '\r\033[K'
    close_seal
}

close_seal() {
    if skipped; then
        git update-index --no-skip-worktree -- "$SETTINGS"
    fi
    git checkout HEAD -- "$SETTINGS"
    echo "seal: closed"
}

case "${1:-}" in
open) shift; open_seal "$@" ;;
close) close_seal ;;
status) if is_open; then echo "open"; else echo "sealed"; fi ;;
*) die "usage: scripts/seal.sh open [--for <time>] | close | status" ;;
esac
