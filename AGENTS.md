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

## Go version

When schemaf (its codegen, or a dependency it pulls in) needs a newer Go,
upgrade: bump `go` in `go/go.mod` and `go.work` and the image in
`Dockerfile.gen` together. Following schemaf's Go version is mandatory.

## Code generation

After changing `schemaf.toml`, `go/db/migrations/`, `go/db/queries/` or the
endpoint structs in `go/api/*.go`, run `./schemaf.sh codegen`. It writes
`go/api/endpoints.gen.go`, `gen/openapi.json` and the generated clients; the
OpenAPI spec comes from the endpoint structs and their doc comments.
