# Releasing

A release is what instances install: each carries a signed artifact, and an
instance's admin upgrades to it from the app (`deploy/README.md`). The owner
cuts releases from a laptop, signed in with `gh`:

```sh
make prerelease   # a prerelease (2.0.0-dev.N) of v2/development
make release      # v2/development into v2/main, back-merged, then a release of v2/main
```

Prereleases reach the instances on the `prerelease` channel, such as the dev
instance; releases reach every instance. `scripts/release.sh <branch>
--dry-run` shows the next version and its notes without releasing.

## What happens

1. `scripts/release.sh` takes the branch as origin has it: in place when it is
   checked out and clean (a checkout behind origin is offered a fast-forward
   pull), otherwise in a temporary worktree. It runs semantic-release with the
   `gh` token, which picks the next version from the conventional commits
   since the last release (`.releaserc.json`), creates the GitHub release with
   the notes and pushes the tag `v<version>`. A release comments on the issues and pull requests it
   ships and labels them; a prerelease does neither.
2. The tag starts `.github/workflows/release.gen.yml` (written by schemaf's
   codegen). It builds the artifact `chatgamelab-<version>.run`, signs it with
   a keyless Sigstore attestation, attaches both to the release and rebuilds
   `releases.json`, the index every instance reads.
3. Within the hour, each instance on the release's channel fetches and
   verifies it, and its admins see it under Server settings › Updates, with
   its notes.

A range with no releasable commit (only `docs:`, `chore:`, …) gives no
release. A failed `release.gen.yml` run leaves a release without artifact,
which no instance offers; re-run it from the Actions tab.
