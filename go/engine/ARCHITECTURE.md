# Engine architecture

ChatGameLab's game engine: a Go module that runs one AI-driven game session,
embedded in the platform or served on its own. The platform owns accounts,
workshops, sharing and authoring; the engine owns everything from a resolved
session spec onwards.

This document sits between the spec and the code. The spec
(`docs/chatgamelab.typ`) says what the platform does; this outlines how the
engine is built to do it; the code holds the details.

## Blocks

The engine ships a library of **building blocks**, each wrapping one mechanism
and configured by prompt into whatever role a genre needs:

| Block | Mechanism | Roles |
| --- | --- | --- |
| **Tool call** | one single-shot text-in/text-out call | rephrase, condense, translate, observer |
| **Tool call that takes audio** | one clip of speech in, a structured result out | rephrase of a spoken action |
| **Threaded call** | a call on a continuing conversation, spoken while narration is on | Expand, sharing Outline's thread, and narration |
| **Threaded call with a structured result** | a threaded call whose reply is JSON of a schema | Outline: its outline, image prompt and status |
| **Image generation** | prompt in, image out | scene illustration, portrait |
| **Live session** | a long-lived, full-duplex speech-to-speech connection | NPC-Live's conversation |

A new short transformation is a prompt on an existing block, which keeps the
library small. Player inputs are source nodes and outputs are sink blocks
without a model call, one node per stream, so a genre's event schema reads
straight off its graph.

## Wiring

A genre is **Go code that wires blocks together**: a small, fixed, compiled-in
set, written by hand.

- **Ports are typed by the data they carry**, and the compiler checks every
  edge.
- **The wiring defines the transport.** Streamed text is a port kind of its
  own, apart from whole text: a block taking whole text waits for the whole, a
  block taking the stream gets each piece. Adventure's Expand streams the
  player's text, so the prose appears as it is written; while narration is on
  its turns are spoken, and the audio reaches the player beside the text, as
  it comes.
- **Fan-out** gives each consumer its own stream; **a cycle is a back edge**
  like any other. NPC-Live's observer loop is one.
- **A per-genre test checks completeness**: every required input and output is
  wired, and every block is reachable from an input.
- **A failure is a block's, never a turn's.** Every block that makes a model
  call has a failure output: after its retries, a failed call sends the input
  it was given there, and nothing on its normal outputs; on a shared thread it
  hands the thread's turn on. The genre wires each failure output to an error
  sink built with a code, which tells the player and lets the game go on; to
  the problem sink, which ends the session; or to any block that takes text.
  The completeness check refuses a failure output left unwired. The provider's
  reason stays on the block's state report, for the graph view, which draws
  failure edges apart as a pale red line.
- Two genres justify neither a DSL nor a generic scheduler, so the graph stays
  Go and keeps its type safety.

### Init, gate, loop

A genre's graph is **P-shaped**: an init stem that runs once, a **gate**, and
the loop. The gate waits for the blocks wired into it to report
working-then-ready and sends the cue that opens the game. What gates is a
wiring decision: NPC-Live gates on its portrait and releases the player's
voice with the gate; Adventure gates on its preparation and releases the
player's input once the opening's prose is done.

Every block reports its state changes, and every report reaches the client,
which draws the wiring once and then tracks phases.

The drawing is placed by hand. Each genre keeps a layout beside its wiring
(`adventure_layout.go`, `npclive_layout.go`) that gives every block a grid
cell, in a reading order a person chose; the topology carries the cells, and a
test fails when a block has no cell. On a dev server the blocks can be dragged
to other cells and the layout copied back as Go source.

## Genres

### Adventure

**Rephrase → outline → expand**, because a single-shot "what happens next" call
is too willing to do what the player asked. Rephrase turns the input into third
person with an uncertain outcome; Outline models a plausible world reacting to
it; Expand turns the outline into prose.

- **Preparation** runs two tool calls in parallel before the gate opens:
  condensing a scenario over 500 characters, and adapting the image style to
  the youth-protection rule. Both are skipped when no offer allows pictures;
  either failing ends the start. Their results are kept, so a resumed session
  never makes them again.
- **The world has the initiative.** The gate sends the Initialization Prompt
  straight to Outline, skipping Rephrase, and the opening runs through Outline
  and Expand into narration, a picture and the initial status. The opening's
  dice are rolled at launch and stored with the spec.
- **The picture prompt** is built without a model call from one reply's image
  prompt and outline, the game's description, the condensed or full scenario
  and the adapted style, inside the scene frame.
- **The player types or speaks**, on two inputs always wired. A typed line
  goes to Rephrase, and at once to the player's line on the `said` stream. A
  spoken clip goes to Rephrase's spoken twin on the hearing model, whose reply
  is a schema the genre owns: what was heard, the player's words verbatim in
  their language, for the player's line, and the action, rephrased as
  Rephrase does, for Outline. The `said` stream is kept in the history like
  the prose, so a reload replays both speakers.
- **A failed call during play** costs the turn: Rephrase, Outline and Expand
  lead to the `turn_failed` error sink, the picture to `image_failed`. A clip
  of which nothing came through, the spoken Rephrase failing or hearing no
  words, leads to `voice_failed` and starts no turn.

```mermaid
graph TB
    TYPED["Player input text"]
    SPOKEN["Player input audio
    (one clip per press)"]
    PREPROCESS(["Rephrase
    (tooling model)"])
    HEAR(["Rephrase audio
    (hearing model, takes audio)"])
    SAID(("Player's line
    (said)"))

    TYPED --> PREPROCESS
    TYPED --> SAID
    SPOKEN --> HEAR
    HEAR -- heard --> SAID

    OUTLINE(["Outline
    (story model — starts/continues the thread)"])
    PREPROCESS --> OUTLINE
    HEAR -- action --> OUTLINE

    OUTTEXT["outline text"]
    PROPS["properties map"]
    IMGPROMPT["image prompt"]
    OUTLINE --> OUTTEXT
    OUTLINE --> PROPS
    OUTLINE --> IMGPROMPT

    EXPAND(["Expand
    (story model — same thread,
    spoken while narration is on)"])
    IMAGE(["Image
    (image model, optional)"])
    VETO(["Veto / fact-check
    (deferred)"])

    OUTTEXT --> EXPAND
    IMGPROMPT --> IMAGE
    OUTTEXT -.-> VETO

    OUT(("Turn output"))
    EXPAND -- text --> OUT
    EXPAND -. "audio (optional)" .-> OUT
    IMAGE --> OUT
    PROPS --> OUT
```

Both inputs, the image and Expand's audio are always wired. The session's
`images` toggle switches the image block on and off; `audio-in` switches the
spoken Rephrase, so speech is refused while it is off, though a clip already
sent is heard; `audio-out` has Expand's turns spoken, and the story thread runs
on the tier's speech model meanwhile, so the model that writes the prose
speaks it.

### NPC-Live

A portrait is generated and gates the conversation; the loop is currently a
single live session, keeping the player's voice and the reply directly
connected.

Control comes from the side: an **observer** reads the character's transcript,
judges it against the guardrail, for sycophancy and for character, and injects a
correction back into the live session. It needs its own thread, because a
character softens across several exchanges.

The scenario and the guardrail reach the model through separate channels of the
live session, so designer text and the platform's constraint stay apart.

## AI platforms

The engine publishes a **catalogue** of the AI platforms it runs: what a person
needs to understand when choosing a platform and a tier. Its settings are
written into the engine.

- **A platform** has an id, a name, whether it accepts a user's own API key, and
  four **toggles**, each saying whether a feature exists on it: `images`,
  `audio-out` (narration), `audio-in` (voice input) and `audio-live`
  (full-duplex speech).
- **A genre declares the toggles it needs**, so the genres a platform can run
  follow from its toggles. NPC-Live needs `audio-live` and `images`; a launch
  on a platform that lacks one is refused.
- **A tier decides quality, never whether a feature exists.** The tiers are
  `low`, `medium` and `high`, labelled Economy, Balanced and High; `max` is
  read as `high`.
- **Each tier lists a model per model type**, with the parameters it is called
  with, such as a picture's size and quality or a voice's name.
  `tooling` and `story` are always listed; each toggle that is on adds one:

| Toggle | Model type | Used by |
| --- | --- | --- |
| — | **tooling** | rephrase, translate, observer |
| — | **story** | Outline and Expand, on one continuing conversation |
| `images` | **image** | image generation |
| `audio-out` | **speech** | the story model, speaking: Outline and Expand while narration is on |
| `audio-in` | **hearing** | the tool call that takes audio: Rephrase of a spoken action |
| `audio-live` | **live** | the live-session block |

Tiers have capabilities and model choices: a model and its parameters are set
in the tier table and nowhere else. Beside the tiers, each platform has a
**model table**, keyed by model id, saying what each model is: the kinds it
takes and makes (text, audio, image) and its price, dated, with its source. A
test requires an entry for every model a tier names, and that each model takes
and makes what its type needs, such as the hearing model audio.

- **An offer** is one way a session may run: a platform, a key and the
  **caveats** on using it, named by a unique ID such as the shared key's.
  Everything names an offer by its ID, never by position. Caveats select what
  is allowed and never change a model or a parameter.
- **Caveats** are an allowance, the toggles allowed at each tier the offer may
  be used at; a default tier and toggles inside it; and whether the player may
  customize.
- **Settling** picks the session's offer, tier and toggles from the offers and
  the player's preference. With customization off the default applies; with it
  on the player chooses within the allowance, less than the default included.
  Whatever is settled must meet the genre's needs; a session never changes
  platform.
- **The preference is the player's wish**: a platform, an offer's ID, a tier
  and toggles. It is persisted with the spec and fitted to the current
  offers on every settle, since a share can change during a game: the same
  offer or another on the same platform, the nearest allowed tier, toggles no
  longer allowed switched off. A choice sent from the options dialog that lies
  outside the caveats is refused instead.
- **Toggles are switches in a fixed graph.** The settled toggles are their
  starting positions; during play a toggle may go off at any time and on when
  the offer allows it at the session's tier. A block whose toggle is off does
  no work and bills nothing, and what it made before stays.
- **`engine.Validate`** reports a spec's errors and warnings, each with a field
  path. Launch and resume refuse any error; a platform may call it ahead of a
  launch.

The catalogue is exposed as `engine.Platforms()`, as the transport's
`GET platforms`, and printed by `cmd/models`. Beside it, `GET genres` lists each
genre with the toggles it needs, and `GET themes` the playback-engine's looks,
for the game editor.

- **`engine.CheckOffer`** asks an offer's provider whether it accepts the key,
  with a request that costs nothing, and tells a rejected key from a provider
  that does not answer. The platform calls it before creating a session; the
  mock always passes.

Conversation continuity — the thread, the live conversation — is per-block
state, keyed by the block's name. A thread several blocks share, as Outline
and Expand share the story model's, is exported once, under its own name.

### Threads

A threaded call continues a conversation the provider holds: OpenAI's
Responses API names the previous response, Mistral's Conversations API
appends to a stored conversation. The engine keeps the continuation, the
model and offer it was made on, and the whole transcript. A continuation that
cannot be used, on another offer or model, past OpenAI's 30-day retention, or
one the provider no longer knows, gives way to a fresh provider thread seeded
with the transcript, so a session never ends because a provider forgot its
thread.

- **One call at a time**, in turn: the next turn's Outline waits for Expand.
- **The System Message is sent once**, as the thread's first message: on
  OpenAI a developer message at the start of the first call's input, since
  OpenAI carries no `instructions` over a previous response while input
  messages are stored; on Mistral the conversation's instructions as it
  starts. A fresh provider thread seeded from the transcript begins with it
  again. Adventure's System Message is the game engine prompt, the scenario,
  how the game starts, and the youth-protection rule.
- **Each call's message is a block's**: the player's action, or the game's
  own message, such as the Initialization Prompt and Expand's prompt, sent as
  a developer message on OpenAI and as the user's on Mistral.
- **The youth-protection rule travels with the calls**, worded "NARRATION
  RULES must be respected: …": on OpenAI in a developer message of its own,
  right before the call's message, so designer text never shares it; on
  Mistral, which has no developer role, appended to the user message. How
  often a threaded block's calls carry it is its **constraint frequency**,
  from 1, every call, to 10, counted per block from the transcript; the
  System Message and the opening always carry it. A spec sets it by block
  name, and it may change during play, stored with the spec.
- **On Mistral**, appending takes neither a model nor instructions, and an
  input may only be a user's or an assistant's message. A conversation stays
  on the model it started with.
- **A stream that breaks partway** is made again from the same continuation,
  as the provider never completed that response; the pieces already sent are
  dropped first. An exhausted retry or a hard failure fails the call, and the
  block's failure edge decides what follows.
- **Expand's prose is tidied** once finished: prose taken out of JSON, a JSON
  block cut off its end, markdown emphasis dropped. Prose that changed
  replaces the open utterance; a reply with no prose left is made again, up
  to three times.
- **A turn may be structured**: the thread adapter takes an optional JSON
  schema, sent in the provider's strict mode, so one thread carries
  Outline's structured turns and Expand's plain ones. Every call states its
  format, a plain one as text, so a format never carries over to the next
  turn.
- **A plain turn may be spoken**: the model says its reply, the transcript
  streams as text and the audio, 24 kHz PCM16, beside it. The reply the
  thread keeps is the transcript, so the transcript stays text. A thread on a
  model that cannot speak answers in text.
- **The thread's model follows narration.** While `audio-out` is on, the
  story thread runs on the tier's speech model, off on its story model. A
  switch applies from the next call: the continuation made on the other model
  is dropped, and a fresh provider thread is seeded with the transcript, for
  one uncached call per switch.
- **On OpenAI's speaking models**, gpt-realtime, the thread runs over the
  Realtime API's server-side WebSocket. The continuation is the session, held
  open between calls; the session's 60-minute limit, a dropped connection or
  an engine restart reseed a fresh one from the transcript. The System Message
  is the session's instructions; the youth-protection rule and the game's own
  message are system messages, the player's a user message, and a response
  sets no instructions of its own. Realtime has no JSON schema format, so a
  structured turn is a forced function call, named after the schema with the
  schema as its parameters, and the caller checks its arguments against the
  schema; a reply not kept is deleted from the session.

### A structured result

Outline's schema belongs to the genre, a call parameter no spec overrides:
the outline, the image prompt, and the status, one property per status field
of the session, every one in every reply. Each call sends the action with the
current status values, so the thread's memory is not trusted for them.

- **Parsing is part of the call.** A reply is tidied first: a markdown fence
  and any text around the JSON object go. One that still does not parse
  against the schema, a refusal or one cut short is a bad reply: it is never
  kept on the thread, and the call is made again from the same continuation.
- **Three bad replies fail the call.** Nothing goes to Expand, the image or
  the status; the thread hands the turn past Expand; the block is ready again;
  and its failure edge leads to the `turn_failed` error, with the line the
  player sent back in the input box. This budget is apart from the network's
  retries.
- **`cmd/probe`** (`make probe`) measures, on real keys and by hand, how
  reliably each platform's story models answer a structured turn, streamed,
  fresh, appended and on a fresh thread seeded with the transcript: valid,
  invalid, refused, cut short, failed, whether it streamed, latency and cost.
  For the speaking models it also follows spoken turns: the lag between
  transcript and audio, and whether earlier audio inflates the input. With
  `-hearing` it has the hearing models hear clips of speech and silence: valid
  replies, how faithful "heard" is, silence heard as nothing, latency and cost
  per spoken minute. It gives stats, no verdict.

## The session spec

A session launches from a spec and the offers it may run on; the engine
resolves nothing against platform data.

- **Scenario and guardrail** — the designer's text and the platform's
  youth-protection constraint, already resolved.
- **Starting message and description** — how the game starts, in the
  designer's words, and the game's idea in a sentence.
- **Dice and constraint frequencies** — Adventure's opening values, rolled at
  launch, and how often each threaded block repeats the youth-protection
  rule.
- **Genre prompts** — the mechanics, belonging to the genre as a named map. A
  spec may replace any of them.
- **Genre, title, image style.**
- **Theme** — the playback-engine's look, one of its presets; empty is the
  default look. Listed by `GET themes`.
- **Language** — the player's, fixed at creation and carried to the
  playback-engine.
- **The player's preference** — the wish described above, naming an offer by
  its ID, once the player has chosen.
- **The voice.**

The spec is what is stored. **Offers are live inputs** beside it, like
persistence: `Launch`, `Resume` and `Validate` take them as an argument, and
`Adapt` replaces them. Each is a platform, a key provider and caveats; the key
provider is an injected function, so nothing stored holds a secret and a
rotated key takes effect on the next call. An offer with errors, or without an
ID of its own, is left out with a warning; no usable offer refuses the launch.
Only the standalone launcher and the engine's tests supply the mock's offer.
A launcher's game file keeps the two apart: `{ "spec": …, "offers": […] }`.

The platform assembles a spec from its cascade; a standalone launcher or a third
party assembles one by hand. The engine treats both the same.

## Sessions and persistence

**Auth authorises launching a session; the session id plays it.** The id is a
bearer handle scoped to one session, so an embedded playback-engine can hold it while the
key stays on the server.

- **A session is its spec, its state and its event log.** The spec is fixed at
  launch, apart from the player's preference; the state holds what blocks
  export, such as conversation continuity and status values, and what the
  session has spent.
- **History is the ordered log of what a player reads back**: text, pictures,
  status and errors, replayed through the same client path as live events.
  Live audio, narration, phases and usage are not part of it. Streamed text
  is kept as one event per finished utterance, followed by its close, so a
  reload replays whole utterances; a discarded one is never kept.
- **Storage is injected** as `engine.Persist`: save an event, save the state,
  load the state, load the history. The state is saved at launch, before it
  returns, and at checkpoints while the session runs; each history event as it
  happens. The engine ships a memory store and one that keeps nothing.
- **`Resume` restores a session by its ID** from the loaded state and history,
  on fresh offers. A schema version mismatch, or a block the wiring no longer
  has, refuses the resume as stale. Sessions are disposable; Games and Users
  keep real migrations.
- **A live conversation is not re-attached** after a restart: it comes back
  paused, and the player resumes it with one press from the kept transcript,
  as after idling.
- **A refusal names its problem** — no key, a genre the key cannot run, no
  plays left, a rejected key, an unreachable provider, a stale or unknown
  session — so the playback-engine can say why. A key rejected during play is
  reported on the `problem` stream, distinct from a lost connection.
- **A session may begin by choosing.** With more than one offer, or one the
  player may customize, the graph is built and waits: the player's options
  dialog reads the offers, sends a choice, and the game starts. Nothing runs
  or bills before. A stored preference skips the dialog.
- **Every settlement is announced** on the `settled` stream with the
  preference, so the platform stores it with the spec; an in-game toggle
  change is announced the same way.
- **A running session adapts** when the platform passes it new offers: the
  preference is fitted again, the blocks' adapters are replaced in place, and
  the player is told what changed. An offer on another platform ends the
  session.

## Code boundary and deployment

The engine is a sibling module of the server, and the compiler keeps its
internals out of reach of any other module.

- **Public surface**: a launched or resumed session, driven by plain Go calls
  and read as a stream of events.
- **One HTTP transport, used two ways.** The core serves no HTTP. The
  transport is one schemaf endpoint per route, each configured with the engine,
  at `/api/engine/…` in both uses.
  - **Embedded**: the platform declares an endpoint of its own for every
    engine route, same method, path and types, which applies its access policy
    and calls the engine's endpoint. They appear in its OpenAPI spec and its
    generated clients; a test keeps the two route lists equal.
  - **Standalone**: a small dev server serves the engine's endpoints from a
    hand-assembled spec, on a plain mux, through schemaf's endpoint pipeline.

  The transport deals in sessions; authentication, roles and share links
  belong to whoever serves it.
- **A refusal is schemaf's error response** `{"error", "code"}`, its code the
  problem and its status the problem's: unknown 404, denied 403, stale 410,
  unreachable 502, failed 500, any other 422.
- **Creating goes through the transport, doing the serving side's work.**
  `POST sessions` names a game or a share token, the player's language and a
  one-time key; the serving side supplies the function that launches the
  session (the platform from its game and key cascade, the standalone launcher
  from its game file). A repeat of the one-time key within ten minutes answers
  the same session, and only to the caller who made it: the serving side names
  who is asking, and anyone else is refused without learning the ID.
- **A session not in memory is restored lazily** through a resume function the
  serving side supplies, by the first request that names it.
- **The playback-engine is embedded** in the module, served at
  `/api/engine/playback-engine/`, and calls every route through the platform's
  generated client, based at the page's origin.

chatgamelab.eu runs embedded, so every request passes the platform's checks
before reaching the engine, and a revoked key or withdrawn share takes effect on
the next request despite the frozen spec.

A live conversation runs between the browser and the provider, so a platform
restart drops only the engine's watch over it. A live session is pinned to one
process, which matters once a standalone engine runs more than one replica.

## Usage and cost

Blocks report **cumulative totals** on a port of their own, in four units: text
per token (cached input apart), a speaking or hearing model's audio per token
(in, cached in and out, apart from its text), audio by its length, a live
conversation's open time or a heard clip where the provider bills per minute,
and pictures per picture. The engine prices them from the platform's model
table and reports per block and per model, with a flag for whether every model
was priced.

## Protocol

**The session-scoped event stream is the protocol**, carried according to the
genre's interaction model: a voice genre uses one duplex connection for audio
and events; a turn-based genre reads events as server-sent events, one JSON
event per message, and sends actions as requests. A genre's wired output
sinks define its event types. Utterance boundaries travel in-band: on the
`text` stream an empty value closes the utterance, the discard marker drops
the one still open, and a value starting with the replace marker replaces its
text with the rest; on the `audio` stream an empty frame drops the speech
still playing and queued, so a reply made again speaks from its start. A
`said` value is one whole line of the player's, typed or as heard, which the
playback-engine draws as the player's bubble. A
`props` value is the status values as a JSON object; an `error` value the
game goes on after is a JSON object of a code, such as `turn_failed`,
`image_failed` or `voice_failed`, which the playback-engine says in the
player's language, and the text the failed block was given.

**Push-to-talk sends one clip per press**: the playback-engine records PCM16
at 24 kHz mono while the button is held and sends it on release as one binary
frame on the session's WebSocket, which the engine takes as one utterance. A
recording stops and sends at 30 seconds; a press under 0.3 seconds sends
nothing. A clip refused, as while voice input is off, comes back as the
`voice_failed` error. An adapter wraps the clip as WAV where its provider
wants a file.

**Live audio is browser-direct, watched through a sideband:**

```text
browser ──── WebRTC media + data channel ────> provider
   │                                              │
   └── connection offer ──> engine ── sideband ───┘
```

The engine brokers the connection and attaches to the session separately,
receiving transcripts, usage and close reasons and sending control events. The
key and the provider's session id stay on the server.

**An idle conversation is closed**, since a live session bills while open, and
resumed on request: from a fork of the provider's stored session, or from a
fresh session seeded with the transcript the engine kept.

A reloading page rebuilds from the session's topology, state and history.

## Frontend

The playback-engine is **plain HTML, CSS and TypeScript, with no framework**: a game turn
is a timeline of concurrently timed streams, which wants imperative sequencing.
A dependency is welcome when it takes data and paints, and kept out when it
wants to own when things render.

- **The headless core is the playback-engine.** It owns the whole session state, so a
  full game plays without a browser; the UI paints that state and pipes input
  back. Spoken audio is handed to a speaker, which the page supplies and plays
  each frame through as it arrives, back to back; the headless core counts the
  frames.
- **The build output is committed** and embedded, so building the engine needs
  no JavaScript toolchain.
- **One playback-engine** serves the platform and every embedder, through the same
  iframe.
- **The playback-engine starts and resumes sessions itself.** Opened with a
  session ID it resumes that one; opened with `new` and a game or a share token
  it creates one. It then replaces its own URL with the session's, without a
  reload, and tells a same-origin parent page by
  `postMessage({ type: "session", id })`. It remembers the session in its own
  browser storage per game or share, so a host page reloaded with `new` offers
  to continue or start anew. Every launch error is shown on its launch panel,
  with a way to try again or start anew. The headless playback-engine follows the same
  flow.

## Testing

A mock platform implements every adapter with canned, paced output, so genre and
pipeline tests run in-process with no key or network, and it can script a
drifting character for the observer to catch. An integration test plays a
session end to end with the headless playback-engine.

## Build state

### Built

- The block graph: typed ports, the gate, state reporting, per-genre validity
  tests.
- NPC-Live end to end on real adapters: portrait, live session with sideband,
  observer, idle close and resume, cost reporting.
- The HTTP transport, the standalone launcher, and the headless playback-engine with its
  instrumentation view.
- The mock platform.
- Persistence, resume with schema-version refusal, the create endpoint and
  the key check.
- Embedded in the platform: it mounts the transport behind its access policy,
  creates sessions from its key cascade and stores them on its tables.
- The tool call, and the threaded call on OpenAI and Mistral: Adventure's
  Outline answers its structured result and Expand streams its prose, both on
  the story model's thread.
- Adventure's prompts: the game master's system message,
  rephrase, the opening, the two narration prompts, the scene frame and the
  two preparation prompts, filled in for the session when the wiring is built.
- Narration: Expand's turns spoken on OpenAI's gpt-realtime models over the
  Realtime API, the story thread switching model with `audio-out`, and the
  playback-engine playing the audio as it comes.
- Adventure wired to its spec: the System Message once, the youth-protection
  rule with every call at its frequency, preparation, the world's opening with
  its dice, the real image block with its picture prompt, tidied replies, and
  failure edges on every block that calls a model.
- Voice input on Adventure: push-to-talk in the playback-engine, one clip per
  press over the session's connection, heard and rephrased in one call on
  OpenAI's gpt-audio-mini or Mistral's Voxtral, the player's line on `said`.

### Missing

- **The observer's thread** — it judges each line on its own.
- **Re-attaching to a live conversation** after a platform restart.
- **Locking down the browser's data channel** to the commands the playback-engine needs.
- **A veto block**, once the spec settles whether a turn can be rejected.
- **Delegation** — a backend model beside the live conversation, where
  outline-and-expand could run behind a voice. Fixed at session creation, and
  billed on top of the voice minutes.
- **Observer cadence** — streamed deltas or completed replies.
