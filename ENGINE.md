# ENGINE.md — Splitting the Game Engine from the Platform

This is an evaluation document: a technical implementation plan for extracting ChatGameLab's
game engine from the platform (accounts, organisations, workshops, sharing, game authoring) into
a standalone, reusable, embeddable capsule. It is not a pitch — there's no cost or timeline
analysis here, no deadline, and no build has started. It exists so the shape of the work is
written down before deciding whether to do it.

## Why

A second project needs a similar chat-driven, AI-generated interactive experience — dialogues
with a single character on a single topic, no workshops, no sharing, no organisational structure.
Building that as a fork of ChatGameLab would duplicate the engine and diverge over time. Building
it by hacking a second flow into the current, deeply platform-entangled engine would make both
worse. The alternative: pull the actual game engine out into something the platform consumes the
same way any other embedder would — including, eventually, that second project, or any other site
that wants to embed a single AI-driven game with no community features attached.

A useful side effect, not the goal: the platform itself gets simpler once the engine's complexity
moves out of it.

## Scope

- Two genres: **Adventure** (today's only genre, rebuilt from generic building blocks) and
  **NPC-Dialogue** (the new one, the actual reason this work exists). No others are currently
  planned.
- Genres are a small, fixed, hand-built set — not a DSL, not user-authorable, not configurable
  beyond the prompt and stage-toggle level described below. Every attempt to over-generalize this
  ends the same way: reinventing a programming language. Two genres do not justify that cost.
- This document does not cover the platform's business logic (accounts, permissions, workshops,
  sharing, jugendschutz cascade resolution) except where the engine's boundary touches it.

## The shared pipeline

All genres run through **one generic, hardcoded pipeline** — not N separate hand-rolled flows.
A genre is realized by configuring which optional stages are active and which concrete building
blocks are plugged into which slots, not by writing its own orchestration from scratch.

The pipeline's shape was arrived at empirically, not by generalizing on principle: an earlier,
single-shot version of the "generate what happens next" step showed real problems — it was too
sycophantic, too willing to just do what the player asked. Splitting it into rephrase → outline →
expand fixed that, independent of whatever else runs in parallel:

- **Rephrase** turns the player's raw input into third person with an uncertain outcome, which
  distances the AI from the player's literal will and keeps it in control of the fiction rather
  than the player dictating outcomes.
- **Outline** is told to model a plausible world reacting to what the (rephrased) player
  character does — "you're writing a book, this is what the protagonist does" — which produces a
  more grounded, less compliant plot than asking for the final prose directly.
- **Expand** then enriches that plausible-but-terse outline into full narrative prose.

This three-step shape is universal across genres for quality reasons alone, independent of
whether a genre ever generates an image.

```mermaid
graph TB
    INPUT["Player input
    (text and/or audio)"]
    CAP{"Configured adapter
    accepts audio input?"}
    TRANSCRIBE(["Transcribe
    (tool adapter)"])
    PREPROCESS(["Preprocess / Rephrase
    (tool adapter, pluggable)"])

    INPUT --> CAP
    CAP -- no --> TRANSCRIBE --> PREPROCESS
    CAP -- yes --> PREPROCESS

    OUTLINE(["Outline
    (plot/prose adapter — starts/continues the thread)"])
    PREPROCESS --> OUTLINE

    OUTTEXT["outline text"]
    PROPS["properties map"]
    IMGPROMPT["image prompt"]
    OUTLINE --> OUTTEXT
    OUTLINE --> PROPS
    OUTLINE --> IMGPROMPT

    EXPAND(["Expand
    (plot/prose adapter — same thread, always runs)"])
    IMAGE(["Image
    (image adapter, single-shot, optional per genre/config)"])
    VETO(["Veto / fact-check
    (future extension point — not built now)"])

    OUTTEXT --> EXPAND
    IMGPROMPT --> IMAGE
    OUTTEXT -.-> VETO

    AUDIO(["Audio / TTS
    (audio adapter, single-shot, optional per genre/config)"])
    EXPAND --> AUDIO

    OUT(("Turn output"))
    EXPAND --> OUT
    IMAGE --> OUT
    AUDIO --> OUT
    PROPS --> OUT
```

Notes on this diagram:

- **The Transcribe branch is capability-driven, not genre-driven.** Whether Transcribe runs at
  all depends on whether the *adapter currently plugged into the Preprocess slot* accepts audio
  input directly. That's a property of whichever model got resolved into that role, checked once
  by the pipeline itself — not a per-genre toggle, and not something a genre's config decides. If
  a future tool-tier model supports audio input directly, the pipeline stops running a separate
  Transcribe call automatically, with no genre code changing at all. This is the first instance of
  adapters declaring capabilities the pipeline branches on; it's plausible other stages will want
  the same pattern later, but nothing else uses it yet.
- **Image and Audio are optional per genre/config** — the fan-out doesn't run every branch for
  every genre. Whether a genre generates a scene image or narrates its text is a stage toggle, not
  a different pipeline.
- **Veto/fact-check is explicitly not being built now.** It's noted so the pipeline's shape
  doesn't accidentally preclude it later: a block whose output is pass/fail rather than content,
  hanging off the same outline Expand and Image consume, with the authority to reject a turn and
  force regeneration (e.g. catching a plot that asserts a wrong fact). Worth designing *for*, not
  designing *now*.

## Building blocks

Every building block implements one generic interface:

- **Input:** text and/or audio
- **Config:** a set of prompts
- **Execution:** a preconfigured AI adapter
- **Output:** text and/or image and/or audio and/or a structured properties map

Confirmed blocks: Preprocess/Rephrase, Transcribe, Outline, Expand, Image, Audio (TTS),
Translate, Theme generation, Status/properties tracking. Each is a purpose-built Go struct against
that shared interface — genuinely a single interface, not a family of loosely related ones, tried
in earnest rather than abandoned at the first awkward fit. Status/properties and theme generation
are the two blocks whose fit is least obvious on paper and worth validating first during
implementation.

Two things this deliberately is **not**:

- **Not a shared mutable state blob.** A block does not take or return "the whole turn so far" and
  write into whatever fields it likes — that makes every block's true dependencies invisible and
  every test require faking an entire turn's worth of state. Blocks stay narrowly typed to exactly
  what they need; the pipeline's orchestration code is what explicitly reads a value out of the
  in-flight turn record and writes a block's result back into it.
- **Not a generic block-execution engine.** There is no scheduler that walks a declarative graph
  of blocks and runs whatever's ready — that's a DSL by another name, and there are only two
  genres to justify it for. The one shared pipeline is itself just hardcoded Go, including its
  parallel fan-out (goroutines/`errgroup`), and it's the same three lines of orchestration whether
  a genre uses it directly or a hypothetical third genre reuses it too.

## AI adapters

Building blocks don't share one AI adapter — they use a **bundle of four adapter roles**, each
resolving to its own model/tier under the session's one platform and API key:

| Role | Used by | Continuity |
|---|---|---|
| **Tool** | Preprocess/Rephrase, Transcribe, Translate, condense-scenario | Single-shot |
| **Plot/Prose** | Outline, Expand | **Threaded** — one continuing conversation |
| **Image** | Image generation | Single-shot |
| **Audio** | TTS, transcription | Single-shot |

This was verified against the current implementation, not assumed:

- `ToolQuery(ctx, apiKey, prompt)` takes no session at all — nothing to thread, confirming it's
  single-shot by construction.
- `ExecuteAction` (Outline) and `ExpandStory` (Expand) both read a `ResponseID` out of
  `session.AiSession`, pass it as `PreviousResponseID`, and write the new response ID back after
  the call. `ExpandStory` picks up the exact ID `ExecuteAction` just wrote — this is **one shared
  thread across both calls**, not two independent ones. Matches the documented "alternating
  phases" behaviour (a JSON phase, then a NARRATE phase, same conversation).
- `GenerateImage` never touches `AiSession` in either provider implementation; Mistral's own
  implementation says so explicitly in its doc comment — "a separate one-shot conversation, not
  the game conversation."
- Neither `GenerateAudio` (TTS) nor `TranscribeAudio` touch `AiSession` either.

**Consequence for persistence:** because Plot/Prose is a real thread, its continuation token
(response/conversation ID) is state that must survive between calls and across turns — the same
category of thing as a generated image, not a special case. It's persisted through the same
`Persist` checkpoint mechanism described below, and because the pipeline is universal across
genres, that token is a fixed, generic field on the session state, not genre-specific data.

## The bundle

A session is launched from one self-contained bundle. The engine never calls back into platform
data to resolve anything — everything it needs arrives already resolved:

- **Prompts** — per building block, admin-tunable config, not hardcoded strings. This includes
  the jugendschutz constraint text, which is not a distinct field or a special type — it's simply
  one of the prompts, re-injected every turn so the model doesn't drift away from it. Which
  cascade level (workshop / organisation / site-by-age) produced that value is entirely the
  platform's concern, resolved before the bundle is assembled; the engine can't tell a
  cascade-resolved prompt from a hand-authored one, and doesn't need to.
- **AI configuration** — platform and tier resolution for each of the four adapter roles.
- **Resolved API key** — who pays. Authorizes the *launch*, not ongoing play (see below).

A bundle can be assembled two ways, and the engine can't distinguish which:

1. **Platform-resolved** — the portal's normal session-creation flow runs the full jugendschutz
   cascade and API-key resolution, and drops the results into bundle fields.
2. **Hand-assembled** — a dev launcher, or a third party embedding the engine directly, supplies a
   raw key and directly-specified prompt/constraint values with no ChatGameLab account involved at
   all. Same bundle shape, assembled by hand instead of by a cascade.

## Session & persistence model

**Auth authorizes launching a session, not playing one.** Once launched, the session id is a
self-sufficient bearer handle — whoever holds it can continue the session. This is what makes
iframe embedding safe: the spendable API key never reaches client-side code, only a scoped session
id does, and that id can't be used to launch anything new. Attaching a session to a user is
optional platform-side bookkeeping (a mapping table the platform keeps outside the engine), not a
structural requirement of a session.

**One fixed session-state struct**, covering the union of fields every genre might need — not
every genre populates every field:

- **Strict SQL columns** for whatever the platform needs to relationally query or enforce: id,
  optional user id, optional game id, created-at, ownership/permission fields, and the plot/prose
  thread's continuation token (generic — every genre uses the same threaded slot).
- **JSONB** for genre-specific state nothing outside the engine needs to query relationally: a
  quiz's score/streak/whatever bonus values it invents, an NPC's trust meter, or nothing at all
  for a genre that doesn't need it.

**Schema evolution by invalidation, not migration.** The JSONB payload carries an explicit
schema-version marker (not just "does it unmarshal cleanly" — Go's JSON decoding is too permissive
to catch a semantic change reliably). A version mismatch on load means the session is simply
treated as invalid. This is acceptable specifically because sessions are cheap and disposable —
unlike a `Game` (a teacher's authored scenario) or a `User`, nobody loses anything of lasting value
when an old session can't be read after an engine upgrade; they start a new one. Games and Users
keep real migrations; sessions don't need them.

**Persistence happens at checkpoints during a turn, not once at the end.** This rules out a pure
"state in, state out" function — real intermediate results need saving mid-flight: a generated
image, the plot/prose thread's updated continuation token, a status update. The engine calls a
`Persist`-shaped interface at each of these points; the portal implements it against Postgres, a
dev launcher can implement it as a no-op or in-memory store. This generalizes a pattern that
already exists today — `stream.go`'s `ImageSaver`/`AudioSaver` callbacks already persist media the
moment it's final, with the caller never knowing where it's saved. `Persist` is that same idea,
widened to cover the whole turn.

## Engine/platform code boundary

Two sibling Go modules, joined by `go.work` during development:

- The engine module's internals live almost entirely under its own `internal/` — compiler-
  enforced: `go build` itself refuses an import from outside the module's own tree. Not a
  convention, not a lint rule, not something that depends on review discipline or on a coding
  agent remembering a rule — the code simply won't compile if the boundary is crossed.
- The engine exposes a small, deliberate public API: a `Launch(bundle) (*Session, error)`-shaped
  call, a `NextTurn(session, action) (*Session, *Response, error)`-shaped call, and the `Persist`
  interface described above.
- **The engine knows nothing about HTTP or REST.** No `http.Handler`, no request/response types
  tied to a transport. The portal owns all routing, request parsing, response writing, and SSE
  framing, translating between HTTP and the engine's plain Go calls. This mirrors what already
  exists today: `stream.go` only ever deals in a typed Go channel; `api/routes/sessions_messages.go`
  is the only place that knows SSE exists at all. The split being proposed here is that same
  separation, generalized to the whole engine instead of just the streaming layer.
- Frontend assets (HTML/CSS/compiled TypeScript) are embedded into the engine module via
  `//go:embed` and exposed for the portal to serve at whatever route it chooses — the engine
  doesn't own an HTTP handler for them either, consistent with the point above.
- Because the module and `internal/` boundaries are real from day one, moving the engine into its
  own repository later — if that's ever wanted — is a near-zero-cost move: stop workspace-
  including it, tag a version, point at a git URL. Nothing needs untangling first, because nothing
  was allowed to tangle.

## Transport

Stays REST + SSE. Not WebSockets. The traffic is turn-based and half-duplex end to end, including
the media within a single turn — text, image, and audio chunks are all part of one turn's
progressive response, not independent ongoing channels. A WebSocket would add message framing,
reconnect/backoff handling, and (if this is ever load-balanced across instances) sticky-session
routing, all to save a connection-setup cost that's dwarfed by what it's waiting on: the AI
generation call, which takes seconds, next to which a POST-plus-SSE-handshake's overhead is noise.

Concretely: **POST an action → SSE streams that turn's progressive output → stream closes when the
turn completes.** Message history and finished media are separately available through plain,
stateless GET endpoints — usable on reload, and sufficient on their own for reviewing a past
playthrough with no live session or engine involvement at all. This already exists today
(`GetMessageStatus`, `GetMessageImage`, `GetMessageAudio` alongside `GetMessageStream`); the split
doesn't change it, it just clarifies which half belongs to the engine's turn loop and which half
is ordinary stateless retrieval.

## Frontend

**The player is iframe-embeddable.** A host page sizes the iframe with ordinary CSS — percentage,
flex/grid, `vh`, media queries — and the content inside reflows exactly as if the browser window
itself had resized. No `postMessage` bridge is needed unless content-driven auto-height is wanted
later; an internally-scrolling box is the natural shape for a chat-style feed regardless.

**Not React. Plain HTML5/CSS/TypeScript, no framework.** This is a firm decision, not a stylistic
preference: React's render-as-function-of-state model fights the actual shape of a game turn,
which is a timeline — several concurrently-timed streams (text, image, audio, status transitions,
background animation) that need imperative sequencing, not reactive re-rendering. This has been a
real, lived cost in the current `game-player-v2` implementation (`useStreamingSession`, the
`text-effects/` animation layer, `StatusChangeIndicator`) — a paradigm mismatch, not a preference
for a different library.

**Sizing the rewrite, checked against source rather than assumed.** The natural worry with
dropping React is the animation-heavy surface — background particle effects, per-message text
effects, theme resolution. Read in full against the current `game-player-v2` implementation:

| Area | Lines | Finding |
|---|---|---|
| `BackgroundAnimation.tsx` — particle configs (stars, embers, confetti, hyperspace, …) | ~650 | Plain JSON handed to tsParticles. tsParticles core has a vanilla JS API (`tsParticles.load(id, options)`); this data ports verbatim — `@tsparticles/react` is a thin wrapper, not where the logic lives. |
| `BackgroundAnimation.tsx` — Waves / Sun / Tumbleweed | ~460 | Already pure CSS `@keyframes`, manually injected via `document.createElement("style")`. The React JSX here only builds static markup once. |
| `BackgroundAnimation.tsx` — Matrix rain | ~80 | Already a raw `<canvas>` + `setInterval` + `ResizeObserver`. React only supplies mount/unmount timing. |
| `text-effects/*.tsx` (12 files) | ~740 | Core animation math (scramble, glitch, etc.) is pure functions. Only the `setInterval`-driven `setState` tick wrapper per file — ~15–20 lines each — is genuinely React-shaped, and it *simplifies* once ported: a vanilla tick writes `element.textContent` directly instead of forcing a re-render to do it. |
| `useGameTheme.ts` + `GameThemeContext.tsx` | ~350 | `generateCssVars`, `mergeTheme`, `getStatusEmoji` are pure functions, ported verbatim. Only `createContext`/`useContext`/`Provider` (~30–40 lines) is React-specific — and it doesn't need porting, it needs deleting: Context solves React's prop-drilling problem, which doesn't exist without a component tree. A vanilla version holds one shared theme object every render function reads directly. |

Out of roughly 4,050 surveyed lines, the code that's genuinely React-dependent and needs real
rewriting is concentrated in the text-effects' tick loops — on the order of 150–200 lines. The rest
was already data, already-imperative canvas/DOM code, or a mechanism (Context) that stops being
needed rather than needing a replacement. The "live backgrounds" risk that motivated checking this
turned out to be much smaller than it looked from the outside.

**Genuinely headless.** A core state machine — session lifecycle, streaming accumulation, turn
progression — owns the timeline and has zero rendering opinion. A thin UI layer's only job is to
paint whatever the core's current state says. This mirrors the backend split exactly: one shared
pipeline of narrowly-typed building blocks on the backend, one headless core with a pure render
layer on the frontend — the same principle, applied on both sides of the boundary.

**The portal consumes its own engine through the same iframe boundary any external embedder
would use.** No special-cased "render the player directly" path inside the portal, not even for a
game author previewing their own game. This is what keeps the capsule honest — there is exactly
one player to maintain, not a portal-internal one and a public-embed one that drift apart.

## Testing

**Fully testable standalone.** The existing mock AI platform (`server/game/ai/mock`) already
implements the full adapter interface with deterministic output and no network calls. Once the
engine has no DB and no HTTP dependency, pipeline and genre-configuration logic become fast,
in-process Go tests with the mock adapter swapped in — a real step change from today, where the
entire `testing/` suite is integration-only, spinning up Postgres and the full backend per run via
`testutil.suite`. Genre-level correctness (does a status update apply correctly, does a stage
toggle behave as configured) doesn't need any of that infrastructure at all.

## Rollout

1. Build the engine module as a genuine `go.work` sibling, developed and validated end-to-end
   through a standalone dev launcher — a hand-assembled bundle in, a running session out, zero
   portal wiring — before the portal is touched.
2. **Adventure first** — rebuilt from the generic building blocks and the shared pipeline. This is
   the abstraction's real test: it's shaped by one known-good genre, and its actual correctness
   only shows once a second, genuinely different genre is built against it.
3. **NPC-Dialogue second** — the forcing function for this whole document. Expected to expose
   places where the abstraction, having only ever seen Adventure, cut a seam wrong; that's the
   plan working as intended, not a sign step 2 was done badly.
4. Once v2 fully works and is wired into the portal, **v1 is purged entirely — frontend and
   backend both.** No prolonged dual-running. Part of the value here is specifically the
   platform-side simplification that comes from deleting the old, entangled implementation, not
   just having a nicer new one sitting next to it.

No further genres are currently planned beyond these two.

## Deliberately deferred

Noted so they aren't silently forgotten, not because they're expected soon:

- **Veto/fact-check block** — a parallel stage with authority to reject a turn's output (e.g. on a
  factual error), sketched into the pipeline diagram above but not designed in detail or built.
- **Capability-flag pattern beyond audio input** — the Transcribe auto-activation is the first
  instance of an adapter declaring a capability the pipeline branches on. Whether other stages
  want the same pattern is open.
- **Existing v1 `Game` data at cutover** — whether existing production games need a one-time
  conversion script into the new bundle-based config shape, or few enough exist that manual
  re-authoring is simpler, depends on a production game count this document doesn't have visibility
  into.
