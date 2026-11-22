#!/usr/bin/env bash
# Cuts a release of the 2.X line, as the signed-in gh user: v2/main gives a
# release, v2/development a prerelease (2.0.0-dev.N). semantic-release picks
# the version from the conventional commits since the last release, writes the
# notes, creates the GitHub release and pushes the tag; the tag starts
# .github/workflows/release.gen.yml, which attaches the signed artifact.
# See docs/workflows/releasing.md; `make release` and `make prerelease` call it.
#
#   scripts/release.sh <v2/main|v2/development> [--dry-run]
#
# The branch is released as origin has it. A checkout that has the branch
# checked out releases in place (clean and equal to origin); otherwise a
# temporary worktree is used, so any checkout can release.
set -euo pipefail

branch="${1:-}"
shift || true
case "$branch" in
  v2/main|v2/development) ;;
  *) echo "usage: scripts/release.sh <v2/main|v2/development> [--dry-run]" >&2; exit 2 ;;
esac

for tool in gh npx git; do
  command -v "$tool" >/dev/null 2>&1 || { echo "$tool is missing" >&2; exit 1; }
done
gh auth status >/dev/null 2>&1 || { echo "sign in first: gh auth login" >&2; exit 1; }

cd "$(git rev-parse --show-toplevel)"
git fetch --quiet --tags origin "$branch"

if [ "$(git rev-parse --abbrev-ref HEAD)" = "$branch" ]; then
  [ -z "$(git status --porcelain)" ] || { echo "commit or remove your changes first: a release names one commit" >&2; exit 1; }
  if [ "$(git rev-parse HEAD)" != "$(git rev-parse "origin/$branch")" ]; then
    if git merge-base --is-ancestor HEAD "origin/$branch"; then
      behind="$(git rev-list --count "HEAD..origin/$branch")"
      echo "your $branch is $behind commit(s) behind origin/$branch:"
      git -c color.ui=never log --oneline "HEAD..origin/$branch" | sed 's/^/   /'
      printf 'pull them (fast-forward) and release origin/%s? [y/N] ' "$branch"
      read -r reply < /dev/tty || reply=""
      case "$reply" in
        [yY]|[yY][eE][sS]) git merge --quiet --ff-only "origin/$branch" ;;
        *) echo "aborted" >&2; exit 1 ;;
      esac
    else
      echo "your $branch has commits origin/$branch lacks: push them (through a pull request) first" >&2
      exit 1
    fi
  fi
  dir="$PWD"
else
  # The local branch follows origin; a branch checked out elsewhere, or one
  # with commits origin lacks, is refused by git here.
  git fetch --quiet origin "$branch:$branch"
  dir="$(mktemp -d)"
  trap 'git worktree remove --force "$dir" >/dev/null 2>&1 || true' EXIT
  git worktree add --quiet "$dir" "$branch"
fi

cd "$dir"
GITHUB_TOKEN="$(gh auth token)" npx --yes \
  -p semantic-release@24 -p @semantic-release/github@11 \
  semantic-release --no-ci "$@"
