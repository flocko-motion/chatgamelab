# Access refactor: ontology, policy, actions

Draft. Records the design agreed so far for replacing the current permission
layer (`server/db/permissions.go`, `server/db/permissions_game.go` and the
inline role checks around them). Nothing here is built yet.

Goal: a policy file that reads almost like plain English, and an architecture
that makes it impossible to bypass it.

## Why

The current model is one `canAccessX(ctx, userID, op CRUDOperation, …)` per
resource. The real rules outgrew that shape:

- **Policy lives in three places.** The two permission files, about 50 inline
  role checks elsewhere (`api/routes/games_private_share.go`,
  `invites_accept.go`, `users.go`, `db/user_invite*.go`, `db/api_key_shares*.go`,
  `canSeeEmails` in `db/institution.go`, `game/resolve_api_key.go`), and list
  filters in SQL ("filtered in query").
- **CRUD is too coarse.** Play, share, sponsor, see-emails and leave get squeezed
  into `OpRead`/`OpUpdate` or escape the permission files. `canAccessApiKey`
  takes eight positional arguments; `gameID`, `sessionID` and `workshopID` are
  `nil` at every call site, so its sponsorship branch is dead code.
- **No resolved actor.** Every check calls `GetUserByID` again, sometimes twice.
  "Admin?" and "head or staff of institution X?" are hand-written about twenty
  times.
- **Drift has produced bugs:**
  - `canAccessGameSession` (Read, Delete): `RoleHead && Institution != nil`
    suffices. A head of *any* institution can read and delete *every* workshop
    session; the workshop's institution is never compared.
  - Staff scope differs: sessions check `Role.Workshop`, everything else the
    institution.
  - `canAccessInstitutionMembers(OpDelete)` dereferences `*targetUserID`
    without a nil check.
  - Error texts do not always match the logic they guard.
- **Invariants inside authorization.** The last-head rule sits in the permission
  check and again in `RemoveUser`.

## Layers

One direction of dependency: DB → ontology → policy → projection → API.

| Layer | Holds | Must not |
|---|---|---|
| **Ontology** | `Actor`, domain objects (`ontology.Game`), refs (`GameRef`), named collections, relational predicates | decide anything |
| **Policy** (`authz`) | pure functions `(Actor, Ref) → Decision`; constructors of projections and grants | touch the DB or any state |
| **Actions** | load, ask policy, mutate, enforce invariants (last head) | decide access themselves |
| **Wire** (`wire`, today `obj`) | API types only: projections | be used below the API |

### Actor

Resolved once per request in middleware. Carries facts, not decisions:
user ID, admin flag, institution and role, workshop, guest flag. Offers the
predicates that form the policy vocabulary (`IsAdmin`, `ManagesInstitution`,
`IsMemberOfWorkshop`, `Owns`). `Head || Staff` is written once, inside
`ManagesInstitution`.

`Actor` does not answer "may I do X to Y". Most rules depend on the resource's
state, so the rule belongs to neither side.

### Refs

Small structs with what a policy needs about a resource, including derived
facts (`GameRef.Institution` = the institution of the game's workshop). The
loader resolves them, often in the same SQL join. The session bug above is
impossible once the policy must read `SessionRef.Institution`.

### Collections

A list is an entity of its own, not a filtered projection of item rights.
`PublicGames`, `OwnGames`, `WorkshopGames{Workshop, Institution}`: the query
defines membership (ontology), the policy decides who may open the collection.
Lists get slim loaders tailored to their use case; they return list rows, not
tiers.

Invariant: `CanRead(list) ∧ g ∈ list ⇒ CanSeeBasics(g)`. Testable per
collection.

Open: the generic game listing is a union (public ∪ own ∪ workshop ∪ shared)
with search on top. Explicit union of named collections, or a collection type
of its own?

## Projections and tiers

The unit of permission is (entity, projection), not the entity.

```go
type GameBasics struct {
    ID          uuid.UUID
    Title       string
    Description string
}

type Game struct { // the Details tier
    GameBasics            // Details ⊇ Basics
    SystemPrompt string
    Scenes       []Scene
}
```

- Tiers are **semantic** (Basics, Details) and **types**. A type cannot expose
  what it does not have; a new field is invisible until someone places it in a
  tier. Field-level policies would need runtime masking and default to open.
- Embedding makes `CanSeeDetails ⇒ CanSeeBasics` hold by construction.
- Cross-cutting sensitive kinds (contact data such as email) get their own
  semantic policy (`CanSeeContactData`) used by several projections, instead
  of per-field rules.
- The DB loads the full object by default. The policy narrows it by returning
  the allowed type. Restriction is a type question, loading stays simple.
- Writes mirror reads: `GameBasicsUpdate`, `GameUpdate`, each accepted only
  behind the matching policy. Replaces the runtime `canEditAll` switch in
  `games_private_share.go`.

## Decision

One type for predicates and policies, so they compose. Zero value denies.

```go
type Decision struct {
    ok   bool
    rule string     // "owns the game"
    sub  []Decision // children for anyOf/allOf
}

func (d Decision) Allowed() bool { return d.ok }
func (d Decision) Err() error    // generic 403/404 carrying the policy ID

func (a Actor) Owns(g GameRef) Decision {
    return rule("owns the game", g.Owner == a.UserID)
}
```

Fields are unexported: only `authz` can produce `ok: true`.

**Provenance.** An allow carries its witness: the rule that granted it and its
facts. A deny is the absence of any proof (negation as failure), so no single
rule "denied". Inside `allOf` the failing term is nameable; at the top the best
explanation is the near misses: per alternative, which terms were missing.

**Opacity.** The client gets the policy ID (`CanEditGame`) and a generic 403 or
404. The server log gets the full tree. Policy ID plus request ID is enough
for support.

**Coverage.** Logging the granting rule per allowed request shows which rules
fire in production. Rules that never grant are dead or redundant.

## Policy file

Target shape. No control flow, only `anyOf`/`allOf` over named predicates.
Anything that needs an `if` is a missing predicate or a missing fact in a ref.

```go
func CanSeeGameBasics(a Actor, g GameRef) Decision {
    return anyOf(
        a.IsAdmin(),
        a.Owns(g),
        g.IsPublic(),
        a.HasShareLinkFor(g),
        a.IsMemberOfWorkshop(g.Workshop),
        a.ManagesInstitution(g.Institution),
    )
}

func CanSeeGameDetails(a Actor, g GameRef) Decision {
    return anyOf(
        a.IsAdmin(),
        a.Owns(g),
        a.ManagesInstitution(g.Institution),
    )
}

func CanEditGame(a Actor, g GameRef) Decision {
    return CanSeeGameDetails(a, g)
}
```

A rights matrix (actor fixture × resource fixture × policy) can be generated
from this file for review.

## Enforcement

| Gap | Mechanism |
|---|---|
| Output without policy | `httpx.Respond` takes only `wire.Projection` (interface with an unexported method, so only `wire` types satisfy it) |
| Building a projection without policy | Linter: composite literals of `wire` types only inside `authz`. Hard CI gate |
| Writing without policy | DB mutations take a grant (`db.UpdateGame(ctx, authz.EditGrant, …)`); grants are constructible only in `authz` |
| Policy reading DB or state | `depguard`: `authz` must not import `db` |
| Leaking an API key | `Secret` type (below); `Reveal()` only in `game/ai/*` |
| Wrong facts in a ref | Not enforceable. Resolver tests are the trust anchor |
| Check-then-act race on writes | Load, check and write in one transaction, or condition the write on the loaded state |

## Secrets and funding

Keys are separate entities, never shown, resolved lazily at play start or
login and injected into the AI client.

- **Funding** is a selection policy, not an access check: "which key pays for
  this play" through sponsored games, workshop keys, institution shares.
  `ResolveFunding(a, g, sponsors) (PlayFunding, error)` is pure over loaded
  facts. `PlayFunding` is not a `Projection`, so it cannot reach the API. Today
  this lives in `game/resolve_api_key.go` with its own role checks; the dead
  sponsorship branch in `canAccessApiKey` goes.
- **Plaintext** is protected by convention today (`obj/structs.go`:
  `Key string` with `json:"-"`), which does not stop `%+v`, `slog` or error
  strings. Replace with:

```go
type Secret struct{ v string }

func (Secret) String() string               { return "[redacted]" }
func (Secret) GoString() string             { return "[redacted]" }
func (Secret) LogValue() slog.Value         { return slog.StringValue("[redacted]") }
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"[redacted]"`), nil }
func (s Secret) Reveal() string             { return s.v }
```

The `game/ai` interfaces take `Secret` instead of `string`; `Reveal()` is called
only where the HTTP header is set.

## Frontend

- Keeping the JSON shape keeps the runtime unchanged.
- swag names models by package: `obj.Game` → `ObjGame`, `wire.Game` →
  `WireGame`. The frontend uses `ObjGame` 118 times, `ObjUser` 70. Rename in one
  mechanical sed commit together with the package rename.
- Generated fields are all optional today (`cloneCount?: number`). Then
  `GameBasics` is assignable to `Game` too, and the compiler misses Basics data
  flowing into Details components. Run swag with `--requiredByDefault` so the
  subset relation holds only in the right direction. Own commit, before tiers;
  it will surface fields declared without `omitempty` that are sometimes empty.
- Narrowing list endpoints to Basics is a separate step. `tsc` then lists every
  place the UI relies on overexposed data.

## Migration

About 104 handlers and 141 `httpx.WriteJSON(w, status, any)` calls. Strangler
with a ratchet, no big bang.

0. **Session bug fix.** Small commit, independent of the rest.
1. **Foundation, no behaviour change.** `Actor` middleware, `Decision`,
   `wire.Projection`, `httpx.Respond`, `Secret`. CI counter on data-carrying
   `WriteJSON` calls that may only go down.
2. **Package split.** `obj` → `wire` plus frontend sed. Introduce
   `ontology.Game` and move internal uses package by package. Counter on
   `wire` imports outside `api`/`authz`; at zero, enable the literal linter as a
   hard gate.
3. **Record current behaviour** per resource before migrating it: table tests
   against the old `canAccessX`. Decide per row: intended or bug. Bugs change
   explicitly.
4. **Vertical slices**, each deleting its old `canAccessX`:
   1. Sessions: small, holds the known bug, proves the pattern.
   2. Games: the core, most projection value.
   3. Workshops, institutions, users: most inline checks, mostly mechanical.
   4. Invites, API key shares, funding: most domain logic, last.
5. **Narrow exposure**: lists to Basics, `--requiredByDefault`, frontend fixes.

## Open questions

- Generic game listing: union of collections or its own collection type?
- One summary per entity, or per-collection projections (a public list without
  creator data)?
- Where `Actor` resolution lives for guests and share-token users, who today
  bypass workshop checks via `PrivateShareID`.
