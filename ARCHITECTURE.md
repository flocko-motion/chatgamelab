# Architecture

How ChatGameLab is built: the bridge between the spec (`docs/chatgamelab.typ`)
and the code. The engine has its own: `go/engine/ARCHITECTURE.md`.

## Baseline

All code must conform to the built-in `brokkr` linters (`brokkr lint`).
That floor is enforced automatically.

## Project rules

### The secure core decides who may see and change what (#344)

- **Layers.** schemaf establishes who the caller is; the core (`go/core`)
  decides what they may see and change; everything else consumes the core.
- **The caller travels in the ctx.** schemaf's `OnRequest` hook runs
  `core.Resolve` once per request; it builds the caller from the hook's
  `*api.Admission`, the only way to one. Every core function works for the
  caller in its ctx; no code hands a caller around. Handlers pass `ctx` and
  decide nothing.
- **The rules are one table.** `go/core/internal/rules/rules.go`: per role,
  grouped by context, one grant per line as `{Action, Entity, Relation}`. Not
  listed means refused; admin is no exception. Code never checks a role.
- **Contexts are the caller's situation**: always, in or outside a workshop,
  the workshop's settings, still registering, and a suspended organisation
  (its members' only context, so only its grants hold). A grant holds
  in one context; a caller holds the grants of every context they are in.
- **Entities split into sections where visibility differs** (`GameCard`,
  `GameContent`, `GameSettings`). No field is in two sections. A loaded entity
  is its base section; others unlock with `AsX()`, a check of flags the query
  computed, no further lookup.
- **List and Read are separate rights.** Listing may be narrower than reading.
  A `Via` relation (a share link, a session id) opens one entity and is never
  listable.
- **Gates join table and SQL.** `go/core/gates.go` turns grants into a query's
  parameters, one line per grant; each relation is one SQL fragment, written
  once, switched on by its flag. `gates_test.go` fails when the table grants a
  relation no gate opens.
- **Wishes only narrow.** Filters, search, sorting and paging come from the
  caller and narrow what the gates let through. An unknown wish is refused.
- **Every list is one contract.** A list (`core.Lister`: its `ListSpec` and
  its one query) answers a `core.ListQuery` with a `core.Page`, through
  `core.List`, which refuses an unknown wish, naming the known ones. Every
  list endpoint embeds `ListParams` and answers `Page[XView]`; its filters
  and orders are enums in the API, so the generated clients carry them.
- **Rights come with the item.** What the caller may do with an item is a
  `may*` flag on it, computed by the gate the action checks; clients read the
  flags and work out no right themselves.
- **The API owns the wire format.** `go/api/wire` converts the core's types
  into JSON, through their public getters and unlocked sections only. A view
  is flat; `hasX` says which sections it holds.
- **Database access.** Queries live in `go/db/queries/`, the core's as
  `v2_*.sql`. Only the core reads them, through `core/internal/store`.
  Hand-written files in `go/db` are v1; an entity is ported when its file is
  gone.
- **Legacy bridge.** `core.LegacyCaller` and its siblings hand v1 code the
  caller as `obj.User`. Each call marks code not yet ported.
- **Readability over performance.** The core is the code people review for
  rights; it is written to be read.

### Insights are part of playing (#455)

- **Every player has them.** Insights (spec: Insights) show the workings of
  a session to whoever plays it, on the platform as on the standalone
  server, at every screen width. They are the transparency the platform
  teaches with.
- **The secure core decides who sees them.** Inspecting a session is playing
  it and reading its game's content; the session guard (`core.OpenSession`
  with `InspectSession`) checks both. Insights show in production as in dev
  mode.
