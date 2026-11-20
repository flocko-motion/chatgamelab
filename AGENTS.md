# ChatGameLab: notes for agents

## What to read

- `docs/chatgamelab.typ`: the spec, what the product is and does.
- `docs/rules/`: rules beyond the spec. Each names the files it applies to
  (`applies_to`); before changing such a file, read the rules that match it.
- `docs/workflows/`: recipes for reaching a goal.

## Git

- **Commit only what the owner has staged.** Staging is how a change is
  approved. Never stage your own edits, never commit them; leave them unstaged
  and report that they are ready for review.
- Commit messages are conventional commits (`feat:`, `fix:`, `refactor:`, …,
  a scope is fine): semantic-release cuts the versions from them.

## One worktree per issue

Several agents work on this repository at the same time. The main checkout
belongs to the owner and the planner (reviews, commits). **Never implement an
issue in the main checkout.** Each issue gets its own git worktree, so changes
from different issues never mix in one working tree.

**Before creating a worktree, propose exactly this name, branch and base and
wait for confirmation**, e.g. "I'll create `.worktrees/349-web-app/` on branch
`feat/349-web-app` from `origin/v2/development` — OK?".

| What | Pattern | Example |
|---|---|---|
| Worktree directory | `.worktrees/<issue>-<slug>/` in the repo root | `.worktrees/349-web-app/` |
| Branch | `<type>/<issue>-<slug>` | `feat/349-web-app` |
| Base and pull request target | `v2/development`, the default branch | |

- `<issue>` is the GitHub issue number, `<slug>` 2–4 lowercase words from its
  title, `<type>` the commit type the change will get (`feat`, `fix`, `docs`,
  `chore`). Work without an issue uses a descriptive slug without a number.
- After confirmation, from the repo root:
  `git fetch origin && git worktree add -b feat/349-web-app .worktrees/349-web-app origin/v2/development`.
- The 1.X line (`development`, `main`) takes only fixes meant for it; such a
  branch starts from `origin/development` and names it as its base.
- If the worktree or branch exists already, say so and propose continuing in
  it.
- All work for the issue happens inside its worktree. The git rules above hold
  there too; say the worktree path when the work is ready for review.
- Ports: the dev stack uses fixed ports (`schemaf.toml`, 7100–7103), so only
  one worktree runs `./schemaf.sh dev` at a time. Tests and the engine on the
  mock platform run in parallel.
- Only the owner, or an agent the owner asks, removes a worktree after its
  branch is merged.

## Go version

When schemaf (its codegen, or a dependency it pulls in) needs a newer Go,
upgrade: bump `go` in `go/go.mod` and `go.work` and the image in
`Dockerfile.gen` together. Following schemaf's Go version is mandatory.

## Code generation

After changing `schemaf.toml`, `go/db/migrations/`, `go/db/queries/` or the
endpoint structs in `go/api/*.go`, run `./schemaf.sh codegen`. It writes
`go/api/endpoints.gen.go`, `gen/openapi.json` and the generated clients; the
OpenAPI spec comes from the endpoint structs and their doc comments.
