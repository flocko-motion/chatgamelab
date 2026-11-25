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

### Embedding plays in the platform's frame (#340)

- **One frame for every host.** An embedded game is the playback-engine in an
  iframe on the host page; the host page never reaches into it, so it never
  reads a player's conversation. A host styles it through the handed-over
  CSS contract (`docs/embedding.md`).
- **The core decides who may frame it.** The playback-engine page's
  `Content-Security-Policy` comes from `core.EmbedFraming`: `frame-ancestors`
  is `'self'`, or the site of the embed link's verified page on any port
  (`scheme://host:*`) once an admin approved it (the only embedding there
  is); framed elsewhere, `img-src` and `font-src` keep handed-over CSS to the
  platform, `data:` and the verified site. An approval holds for the site:
  only a change of scheme or host withdraws it.
- **The host's yes is an access decision**, so the core asks it
  (`core.EmbedPlay`): it posts to the callback in its own records, follows no
  redirect and gives up after a few seconds.

### Notifications are rules over current data (#578)

- **The facts stay where they are.** A notification type is a rule in the
  core (`core/notification.go`) that reads the current data and compares it
  with the user's last acknowledgement of that type; the database holds
  only that acknowledgement (per user, type and optional subject). Without
  one, the start is the user's role's creation.
- **A type is held through the grant that opens its target**: new
  organisations through listing every organisation, waiting pages through
  approving them, an organisation's page decisions through listing its
  page. The grant table stays as it is.
- **Acknowledging names how far**: an entry carries the time of its newest
  fact, and acknowledging stores that time (at most now), so a fact that
  arrives meanwhile stays news. Acknowledging never moves back.
- **The web app polls**: on load, every minute while the tab is visible and
  after an acknowledgement. The bell renders the server's codes and counts
  as i18n texts.

### Insights are part of playing (#455)

- **Every player has them.** Insights (spec: Insights) show the workings of
  a session to whoever plays it, on the platform as on the standalone
  server, at every screen width. They are the transparency the platform
  teaches with.
- **The secure core decides who sees them.** Inspecting a session is playing
  it and reading its game's content; the session guard (`core.OpenSession`
  with `InspectSession`) checks both. Insights show in production as in dev
  mode.
- **Changing what a session runs on is playing it.** Any player of the
  session edits its prompts (`core.OpenSession` with `PlaySession`). Its
  guardrail only staff and heads edit, in their own session
  (`core.MayEditGuardrail`); the guardrail's route hands the page that right
  as `mayEdit`, and the edit route checks it again.
- **Adapting the prompts before a session starts** is for admins, heads and
  staff (`core.MayPresetPrompts`): a game's view says so as
  `mayPresetPrompts`, the portal's start choice then offers its advanced
  settings beside starting and continuing, and the session's creation
  refuses adapted prompts to anyone else.
