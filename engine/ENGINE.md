# ENGINE.md — Splitting the Game Engine from the Platform

This is an evaluation document: a technical implementation plan for extracting ChatGameLab's
game engine from the platform (accounts, organisations, workshops, sharing, game authoring) into
a standalone, reusable, embeddable capsule. It is not a pitch — there's no cost or timeline
analysis here, no deadline, and no build has started. It exists so the shape of the work is
written down before deciding whether to do it.

## Why

A second project needs a similar AI-generated interactive experience — spoken dialogues with a
single character on a single topic, no workshops, no sharing, no organisational structure.
Building that as a fork of ChatGameLab would duplicate the engine and diverge over time. Building
it by hacking a second flow into the current, deeply platform-entangled engine would make both
worse. The alternative: pull the actual game engine out into something the platform consumes the
same way any other embedder would — including, eventually, that second project, or any other site
that wants to embed a single AI-driven game with no community features attached.

A useful side effect, not the goal: the platform itself gets simpler once the engine's complexity
moves out of it.

## Scope

- Two genres: **Adventure** (today's only genre, rebuilt by wiring building blocks) and
  **NPC-Live** (the new one, the actual reason this work exists — a voice-driven conversation with
  a single character, running on a live speech-to-speech model). No others are currently planned.
- Genres are a small, fixed, hand-built set of Go wirings — not a DSL, not user-authorable, not
  declared in data. Every attempt to over-generalize this ends the same way: reinventing a
  programming language. Two genres do not justify that cost.
- This document does not cover the platform's business logic (accounts, permissions, workshops,
  sharing, jugendschutz cascade resolution) except where the engine's boundary touches it.

## Building blocks

The engine ships a library of **building blocks** — small, configurable primitives, each wrapping
one mechanism. A block is typed by what it does *mechanically*, and configured by prompt into
whatever role a genre needs it to play:

| Block | Mechanism | Roles it gets configured into |
|---|---|---|
| **Tool call** | one single-shot text-in/text-out call | third-person rephrase, condense-scenario, translate image style, any short transformation |
| **Threaded call** | a call on a continuing conversation | Outline and Expand, sharing one thread |
| **Structured extraction** | a call returning a typed properties map | status tracking, theme generation |
| **Image generation** | prompt in, image out | scene illustration |
| **Speech synthesis** | text in, audio out | narration |
| **Transcription** | audio in, text out | voice input on a turn-based genre |
| **Live session** | a long-lived, full-duplex speech-to-speech connection | NPC-Live's entire wiring |

Shaping blocks by mechanism rather than by task is what keeps the library small. "Rephrase the
player's input" and "condense a scenario for image prompts" are the same block with different
prompts, so a new short transformation costs a config entry instead of a new type.

**Live session and the text blocks run on different model families**, and the table's first column
is where that shows. `live-session` is GPT-Live and nothing else: a full-duplex conversation model
with no request/response shape to reuse. The rest run on ordinary GPT models through the Responses
API — which is where Adventure is expected to land, and why the generic processing blocks are worth
keeping unaware that a voice genre exists.

**Output sinks are blocks too** — `output-text`, `output-audio`, `output-image`, `output-status`.
They carry no prompt and run no model call, so they're a second family under the same word. Which
means blocks do *not* share one uniform interface, and shouldn't be made to: once genres are
wiring, a shared signature buys nothing. What the library gives you is a catalogue of pluggable
things.

### Typed ports

A block exposes **named, typed ports** rather than one union-typed input and one union-typed
output. `live-session` alone needs two of each — player audio and instruction injection in, audio
and text out — which a single-slot interface can't express.

Ports are Go interfaces, one pair per data type, and wiring is a method on the graph — a method
rather than a free function because the edge has to be recorded somewhere for the validity check
below to have anything to check:

```go
func (g *Graph) ConnectAudioOut(src AudioOut, dst AudioIn)
func (g *Graph) ConnectTextOut(src TextOut, dst TextIn)
```

```go
g := ports.NewGraph("npc-live")
g.ConnectAudioOut(player, live)
g.ConnectAudioOut(live, outAudio)   // speech to the player
g.ConnectTextOut(live, outText)     // transcript as chat history
g.ConnectTextOut(live, observer)    // same source, second consumer
g.ConnectTextOut(observer, live)    // the back edge: steering instruction
```

**Inputs and output sinks are nodes.** `PlayerInputText`, `PlayerInputAudio` and their dummy
counterparts are sources; `PlayerOutputText`, `PlayerOutputAudio`, `PlayerOutputImage` and
`PlayerOutputProps` are sinks. One node per stream rather than one node with four ports, which is
what makes a genre's event schema readable straight off its graph.

**A source port hands out a fresh stream per call.** This is a hard rule on every block, not an
implementation detail: `TextOutPort()` subscribes, it doesn't return a shared channel. Two
consumers of one shared channel would *steal* from each other — the observer eating half the
player's transcript — and nothing about the types would catch it. Subscription also happens when
the edge is connected rather than when the graph starts, so a source that emits immediately cannot
outrun its consumers.

Where a block genuinely carries two outputs of one type — Outline emits a plot outline and an image
prompt, both text — the second is exposed through a `SecondaryTextOut` interface, since a struct
satisfies any given interface only once. No block is expected to need a third. If one ever does,
that's the signal it's doing too much and should be split, rather than the start of an ordinal
ladder.

Four properties follow, and they're the reason ports are interfaces rather than a
`RegisterPipe(TypeAudioOut, from, to)` call carrying a runtime type tag:

- **The graph is compile-checked.** A block that doesn't implement `AudioOut` can't be passed where
  one is wanted. A runtime tag defers that to session start — which in practice means a workshop.
- **Fan-out is just calling `Connect` twice** on the same source. Each consumer gets its own
  stream, so the latency-critical frontend edge never waits on the slow observer edge. No splitter
  block is needed; one would only be worth adding to give a fork its own name in the graph.
- **Cycles cost nothing.** Blocks exist before any edge is drawn, so a back edge is the same call
  as a forward one. A builder requiring upstream-before-downstream would choke on exactly the loop
  NPC-Live needs.
- **Shared plumbing stays small**, because the interfaces are keyed to data type rather than to
  port role. Buffering, fan-out and any later tracing tap are written once per type — audio, text,
  image, properties — instead of once per port.

A genre's wiring is therefore a directed graph over nodes of three kinds — inputs, AI blocks and
output sinks — and it is **not required to be acyclic**: the observer feedback edge is a cycle on
purpose.

**Validity is split between the compiler and a unit test.** Types settle whether an edge is legal;
they say nothing about whether the graph is complete. Because every pipeline is hardcoded and there
are two of them, a per-genre test can check the rest exhaustively: every required input port has a
source, every block is reachable from an input, and every output block the genre's event schema
advertises is actually wired. A block may also declare **required outputs**, which is the same
check facing the other way: without it a genre could wire a character nobody can hear and still
pass, because every input it needed was satisfied. A cheap test that only stays cheap because pipelines are never
user-authored.

The one check neither catches is **termination of a cycle**. The observer watches NPC output and
injects steering, which produces more NPC output for it to watch — self-sustaining by design, and
capable of running away if a correction itself trips the classifier. A bound on consecutive
interventions belongs in the observer block rather than in the graph.

Ports stay **narrowly typed to exactly what a block needs**. No port ever carries "the whole turn
so far" — that makes a block's true dependencies invisible and every test require faking an entire
turn's worth of state.

There is also **no generic block-execution engine** — no scheduler walking a declarative graph and
running whatever happens to be ready. The graph in a genre's wiring is Go, and that distinction is
load-bearing rather than aesthetic: expressed in Go, the compiler checks it; expressed in data, it
needs a runtime validator and the type safety above evaporates. Two genres do not justify paying
that.

## Genres are wiring

A genre is **Go code that wires blocks together**. Genres are a small, fixed, hand-built set,
written by hand and compiled in — a genre is code, not data, and never user-authorable.

An earlier draft of this document specified the opposite: one generic hardcoded pipeline that every
genre ran through, realized by toggling optional stages on and off. That is abandoned. NPC-Live
broke it immediately, because a live speech-to-speech session decomposes into no stages at all —
it is one connection, held open. A spine that only one of two genres can sit on is not a spine.

What the engine provides is the block library, the `SessionSpec`, the persistence interface and the
protocol. What a genre provides is orchestration — hardcoded Go, including its own parallel fan-out
(goroutines/`errgroup`).

### Adventure

Adventure's wiring is the pipeline the current engine already runs, rebuilt from blocks. Its shape
was arrived at empirically, not by generalizing on principle: an earlier, single-shot version of
the "generate what happens next" step showed real problems — it was too sycophantic, too willing to
just do what the player asked. Splitting it into rephrase → outline → expand fixed that:

- **Rephrase** turns the player's raw input into third person with an uncertain outcome, which
  distances the AI from the player's literal will and keeps it in control of the fiction rather
  than the player dictating outcomes.
- **Outline** is told to model a plausible world reacting to what the (rephrased) player
  character does — "you're writing a book, this is what the protagonist does" — which produces a
  more grounded, less compliant plot than asking for the final prose directly.
- **Expand** then enriches that plausible-but-terse outline into full narrative prose.

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

- **The Transcribe branch is capability-driven.** Whether Transcribe runs at all depends on whether
  the *adapter currently plugged into the Preprocess slot* accepts audio input directly — a
  property of whichever model got resolved into that role, checked once by the wiring itself. If a
  future tool-tier model takes audio directly, the separate Transcribe call stops happening with no
  genre code changing.
- **Image and Audio are optional**, toggled in Adventure's own configuration.
- **Veto/fact-check is explicitly not being built now.** It's noted so the shape doesn't
  accidentally preclude it later: a block whose output is pass/fail rather than content, hanging
  off the same outline Expand and Image consume, with the authority to reject a turn and force
  regeneration. Worth designing *for*, not designing *now*.

### NPC-Live

NPC-Live's wiring puts **nothing in the conversation's path**: one live speech-to-speech session,
opened at launch and held open for the conversation's lifetime. No rephrase, no outline, no expand
— the entire value of a live model is that nothing sits between the player's voice and the reply,
and three sequential calls per turn would destroy exactly that.

The constraint is narrower than it first looks, and worth stating precisely: nothing may sit *in
the path*. Work running *beside* the conversation is a different thing, and GPT-Live is built for
it — see **delegation** under Deliberately deferred.

Control comes from the side instead, as an **observer loop**:

```mermaid
graph LR
    PP["portrait-prompt"]
    IMG(["portrait
    (image adapter)"])
    GATE{{"start-game
    (gate)"}}
    MIC["Player audio"]
    LIVE(["live-session
    (live adapter)"])
    OBS(["observer
    (tool adapter, threaded)"])
    FEI["output-image
    → the character's face"]
    FEA["output-audio
    → player"]
    FET["output-text
    → chat history"]

    PP --> IMG --> FEI
    IMG -- state --> GATE
    GATE -- release --> MIC
    GATE -- "opening cue" --> LIVE
    MIC --> LIVE
    LIVE -- audio --> FEA
    LIVE -- text --> FET
    LIVE -- text --> OBS
    OBS -- "steering instruction" --> LIVE
```

**The scenario and the guardrail are not on this diagram, and that is the point.** A port is a
pipeline: values arriving during play, at moments nobody can predict. Both of those are fixed before
the session exists and never change, which makes them configuration — the same kind of thing as the
voice or the model tier, neither of which is drawn either. Expressing a constant as a stream would
say "watch this, it may change" about something that never will.

Their separation survives anyway, and more robustly than an edge would have made it: they occupy
different fields of the provider's session object, the scenario as the model's `instructions` and
the guardrail as a `developer` message. So no scenario text, whatever a designer writes into it, can
reach the field the platform's constraint holds. Structure rather than discipline.

What *does* flow into the live block is instructions appended while it runs — the gate's opening cue
and the observer's corrections — and they share one port, because the provider treats them
identically and fan-in is what a channel already does.

**The cue waits for the session to say it is running.** Creating a conversation and starting one are
two moments, and the provider marks the second with `session.started` — "running, and accepting
commands". An instruction written between them is one the session never had, and the symptom is a
character who was told to greet whoever arrived and says nothing at all. So the cue is held until
that event and delivered once, to the first conversation: every conversation after it continues that
one, and a character who greeted somebody ten minutes ago should not do it again because their
browser reconnected.

`portrait-prompt` stays a pipeline for the opposite reason. Its value is constant, but what follows
it is not: a prompt becomes a picture, and that processing is a flow. The init stem is drawn because
something happens along it.

There is no typed-input node, because GPT-Live has no event that delivers user text to the
conversation. Text reaches the model only as startup history, as appended context, or through a
delegated backend. A genre whose player cannot speak is therefore a genre this wiring cannot serve,
and the graph says so by having nowhere to type.

The live block's text output goes two places at once. One edge is the player's chat history; the
other feeds an observer that classifies what the NPC just said — is it inside the youth-protection
guardrail, is it turning sycophantic, is it still in character — and injects a correcting
instruction back into the live session when it isn't.

**The player's own speech is not shown and not kept.** Nobody re-reads what they just said aloud.
The transcript nevertheless arrives — GPT-Live streams `session.input_transcript.delta` for the
caller's speech without being asked — so this is a decision about what to do with it rather than
whether to pay for it. It reaches no sink and no persistence. That it is free also settles an
earlier open question: the observer can read the player's half whenever that turns out to help,
since "fine, you may pass" only scores as a concession if someone just demanded passage.

Two notes on the observer:

- **Sycophancy is a trajectory property.** A single NPC line almost never reads as capitulation; a
  character softening shows across several exchanges. So the observer keeps its own thread over
  recent NPC output rather than scoring each chunk cold. Guardrail violations, by contrast, are
  usually visible in one utterance.
- **Two prompt slots, two provenances.** The youth-protection guardrail arrives in the
  `SessionSpec` from the platform's cascade and is not editable by a game designer — collapsing it
  into the designer's own text would hand a game author a lever on youth protection that
  JUGENDSCHUTZ.md deliberately denies them. The designer authors the second slot, the **scenario**:
  the setting and rules for Adventure, and for NPC-Live who the character is and what they must not
  concede. One field, both genres.

### What GPT-Live actually is

The model is **`gpt-live-1`**, and it belongs to a different family from the Realtime API an earlier
draft of this document described. The difference is not cosmetic — endpoint, event names, session
shape and control surface all differ — so the properties below are restated from the GPT-Live
documentation rather than carried over.

GPT-Live is full duplex by construction: it listens while it speaks, and decides many times a
second whether to talk, keep listening, pause or interrupt. It pairs with an optional **backend**
that handles delegated work. NPC-Live uses no backend.

- **The character's speech arrives with its text**, as `session.output_audio.delta` and
  `session.output_transcript.delta`. Transcript fragments carry `start_ms` and `end_ms` on the
  session timeline. This side of the conversation is verbatim.
- **The player's speech is transcribed by default**, as `session.input_transcript.delta`, with the
  same timing fields. No opt-in and no second model.
- **There are no turns.** No event marks the end of an utterance, and the documentation is explicit
  that grouping fragments is the application's job. What ends an utterance is therefore a decision
  this engine makes, not one the API hands it.
- **Standing instructions are fixed at creation**; `session.update` rejects them. Adding to them
  mid-conversation is `session.instructions.append`, which appends rather than replaces — so
  re-injecting the jugendschutz constraint never risks discarding the character along with it.
  Two sibling events carry other kinds of context: `session.thinking.append` for facts the model
  should know without saying, and `session.commentary.append` for material it should say aloud.
- **Voice is settled at creation** and cannot change afterwards. Twenty-two built-in voices, plus
  custom voices trained from a recording.
- **Usage arrives as `session.usage.updated`**, carrying cumulative seconds — a running total that
  must not be summed across events, which is exactly the totals-not-deltas contract this document
  already asks blocks to honour. `session.closed` carries the final figure.
- **A session can end by itself**, and the reason says why: `expired` at the duration limit,
  `connection_lost`, `remote_hangup`, and **`content`** when OpenAI's own safety filter stopped it.
  That last one is youth protection firing inside the model, and it deserves surfacing as itself
  rather than as a generic disconnect.

Three consequences the engine has to design around. **The record is symmetric** — both halves are
the same model's own transcript of a conversation it held, which removes the asymmetry an earlier
draft warned about. **Timing replaces ordering**: fragments carry their interval on the session
timeline, so a conversation is reconstructed from `start_ms` rather than from arrival order or from
a bracket the protocol does not have. And **an utterance boundary has to be derived**: this engine
takes sentence-terminal punctuation together with a minimum gap between fragments, and publishes
the result in-band as the empty text delta the protocol already defines. Deriving it once on the
server rather than twice on each client is what keeps the observer and the display agreeing about
what an utterance was.

Both conditions are needed, and the gaps are generous. Silence alone cuts wherever the speaker drew
breath, which puts half a sentence in one card and half in the next; punctuation alone cuts at every
full stop, which turns one reply into a column of boxes. A character speaking slowly — which a
thoughtful one does, and which is most of the appeal — pauses longer between sentences than a
transcript stream makes obvious, so every gap misjudged short breaks up a reply that was meant to be
read together. Fewer, longer cards are the better failure.

**Holding character is the open question.** Rephrase → outline → expand exists precisely because a
single-shot call was too willing to do what the player asked, and NPC-Live is a single-shot call by
construction. Whether a live model stays in character through a long natural conversation — and
whether the observer can pull it back when it drifts — is unknown, and it is the first thing the
spike has to answer.

## AI adapters

Building blocks don't share one AI adapter — they use a **set of five adapter roles**, each
resolving to its own model/tier under the session's one platform and API key:

Roles are named by mechanism, like the blocks that use them. What a genre calls a role —
Adventure's "plot", say — is a use case and stays in the genre:

| Role | Used by | Continuity |
|---|---|---|
| **Tool** | Preprocess/Rephrase, Transcribe, Translate, condense-scenario | Single-shot |
| **Threaded** | Outline and Expand, which Adventure calls its plot/prose stage | **Threaded** — one continuing conversation |
| **Image** | Image generation | Single-shot |
| **Audio** | TTS, transcription | Single-shot |
| **Live** | NPC-Live's live-session block | **Duplex** — one long-lived bidirectional session |

**Tiers pick the model for each role.** A session names one `ModelTier` — economy, balanced,
premium or max — and the platform's own preset table turns it into a model per role, so the engine
holds no model names of its own. Two escape hatches, both explicit:

- `ModelLive: "gpt-live-1"` pins one role to a named model.
- `ModelThreaded: "$max"` lifts one role to another tier, which is how a cheap session keeps one
  expensive stage.

The `$` sigil is what keeps those unambiguous: no model name starts with it, so a field never has
to be guessed at. A tier may leave a role empty, which means the role is unavailable there — speech
output is a top-tier feature.

**A tier also presets the picture's size and quality**, alongside the model, because all three are
what a session is paying for. No genre chooses them: the engine settles what a picture costs, and a
wiring only says that it wants one.

Economy is where this shows most, and it is a rung people really play on — a workshop is a room of
young people on a school's budget, so every saving is multiplied by the size of the room. It
runs the cheapest image model available, at a quarter the output price of the others, at the
smallest size that model offers and the lowest quality. The rung above it is the first to ask for a
custom landscape size, which is a feature only the newer image models have; economy takes a square
instead. That is the single place where a cheaper tier differs in kind rather than in degree, and
it is the shape of a picture rather than anything about the game.

The first four were verified against the current implementation, not assumed. The Live role has no
v1 counterpart to check against, and is instead written against the GPT-Live documentation
described under NPC-Live above:

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

**Consequence for persistence:** because the Threaded role is a real conversation, its
continuation token (response/conversation ID) is state that must survive between calls and across
turns — the same category of thing as a generated image, not a special case. A live session's
conversation id is the same kind of thing.

It is stored **per block, keyed by the block's name in the wiring**, rather than as one field on
the session. A genre may wire two independent threads, and a single field could not hold both.
Resuming rebuilds the graph and hands each block back what it exported; a stored name the wiring no
longer has is an error rather than something to limp past, because it means the wiring changed
under a stored session — exactly the case that should invalidate it.

## The SessionSpec

A session is launched from one self-contained `SessionSpec`. The engine never calls back into
platform data to resolve anything — everything it needs arrives already resolved:

- **Prompts** — and there are two kinds, written by different people.

  **A game designer's prompts** arrive in the spec: the scenario, and the jugendschutz constraint,
  which is not a distinct field or a special type — it's simply one of them, re-injected during a
  conversation so the model doesn't drift away from it. Which cascade level (workshop /
  organisation / site-by-age) produced that value is entirely the platform's concern, resolved
  before the spec is assembled; the engine can't tell a cascade-resolved prompt from a
  hand-authored one, and doesn't need to.

  **A mechanics designer's prompts** live in the genre's own Go file, because a genre is code. They
  describe how a genre works rather than what any particular game is about: the instruction that
  turns a scenario into a request for a portrait, the conversation policy the provider's prompting
  guide asks every live character to carry, what the observer is looking for. None of them belongs
  to a game, and all of them belong to a version of the engine.

  They are a **named map** rather than string constants, and a spec may replace any of them through
  `PromptOverride`. Three reasons, and only the last is convenience: a view can show what a session
  is actually running on, which on a platform that teaches how AI works is the subject rather than a
  debug aid; a prompt nobody can see is a prompt nobody can review; and tuning one costs a field
  rather than a build. An unknown name is an error rather than a no-op, because overriding by name
  is untyped by construction and a misspelling would otherwise leave a session running on the
  default while its author believed otherwise.
- **Genre** — which wiring runs. Fixed for the session's lifetime.
- **Title** — what the game is called. The engine does nothing with it but hand it back on the
  topology, for a client with a header to put it in: naming a game is the author's business, and a
  player asking what they are playing is not answered by the name of the genre.
- **Image style** — how this game's pictures look, held apart from the scenario because they answer
  different questions: the scenario says who is in the frame, and this says how it is painted. It is
  a designer's field rather than a genre's prompt, since two games on one genre should be able to
  look nothing like each other. A game naming no style is given a default one rather than none: with
  nothing asked for the model chooses, and chooses differently every time, so a session's pictures
  would not look like each other, let alone like the game.
- **AI configuration** — platform and tier resolution for each of the five adapter roles, plus the
  voice, which sits outside the `Model*` family because it is a parameter of a model rather than a
  model. One of GPT-Live's twenty-two built-in names; a custom voice trained from a recording is a
  possibility the engine does not yet reach for. It cannot change once a session has been created.
- **A key provider, not a key** — who pays, supplied as an injected function rather than a field.
  Two reasons. The spec is persisted as a blob, so a key field would write the secret into the
  database once per session, and into anything that logs a spec. And resolving at the point of use
  means a rotated or revoked key takes effect without relaunching, which is what v1 already gets by
  re-resolving every turn.

  The platform injects a function closing over its existing resolver; a standalone launcher injects
  one reading a flag or `~/.chatgamelab/config.yaml`. The engine never learns where a key lives,
  and never holds one longer than a request.

A `SessionSpec` can be assembled two ways, and the engine can't distinguish which:

1. **Platform-resolved** — the portal's normal session-creation flow runs the full jugendschutz
   cascade and API-key resolution, and drops the results into spec fields.
2. **Hand-assembled** — a dev launcher, or a third party embedding the engine directly, supplies a
   raw key and directly-specified prompt/constraint values with no ChatGameLab account involved at
   all. Same spec shape, assembled by hand instead of by a cascade.

## The graph is P-shaped: init, then loop

A genre's wiring has a **stem and a bowl**. The stem runs once — v1 does this already, fusing its
system message, translating the image style and generating the theme before the first turn — and
the bowl is the turn loop, which cycles.

They join at a **gate**, which does three things: it waits for every block whose result the first
turn depends on, it releases the player's inputs, and it sends the first message. A gate that only
waited would be decoration — something that opens after the game has already started is not a gate
— so what it releases and what it says are both wired, and visible on the diagram.

The init stem is a pipeline, not a set of independent tasks, which is why it is expressed as graph
rather than as a lifecycle hook. v1's preparation already has chains: condense the scenario, then
translate it, then fuse it into the system message. Edges express that; parallel callbacks cannot.
It also means init products reach play blocks through ordinary edges — the fused system message is
an input to the outline block, not something handed over out of band — so there is no second
mechanism for moving values around.

**Blocks report their own state** on a port of its own — `ready` or `working`, carrying the name of
the block reporting. Two values are enough: "not working" needs no distinction between
never-started and finished, and a block in the turn loop is never finished anyway. A block reports
its initial phase at startup and then only changes, so a reader sees one `working` and one `ready`
per piece of work.

A block's work is whatever it is doing, and for the live session that is the whole conversation
rather than the moments somebody is speaking: it holds a connection, moves audio both ways and
bills by the second from the moment it opens. So it reports `working` when the conversation is
created and `ready` when it is let go, and the graph shows the one block that costs money by the
second lit for exactly as long as it costs. Lighting it per utterance would draw it at rest between
sentences, identically to a block that has never run.

**The gate waits for working-then-ready**, not for an announcement of completion. That cannot be
satisfied by a block which never started, and it does not rely on a block's data output, where the
first value says it has *started* producing — which for anything streaming is not the same as
having finished. Reporting separately from data also lets a block tell the gate it is finished
while its output goes somewhere else entirely.

The contract that follows: a gating block owes the gate a working-then-ready cycle **even when it
finds it has nothing to do**. A resumed session skipping its image generation still reports one, or
the game never starts. A block whose work fails reports one too, or a single broken optional step
holds the gate shut forever.

**Every report reaches the client**, not only the ones wired to a gate. The engine subscribes to
every block's state port by introspection and puts the changes on the session stream, so a client
fetches the wiring once from `/topology` and then only tracks phases. Gating stays a wiring
decision made with edges; observing the whole graph is not, and should not need ten edges drawn
into a collector node to work.

Volume is not a concern, because blocks report *changes* rather than activity: two events per piece
of work, against thousands of audio chunks.

The report names its block for two reasons. A gate has to tell one reporter from another — a block
in the loop works repeatedly, and a busy one must not stand in for a silent one — and a view
drawing the graph has to know which node lit up. That second reason is what the graph view draws: the phases
describe the loop as well as the stem, and how a block oscillates is characteristic. A live session
works for the length of one reply; an observer flicks through it once per line it judges.

**What gates is a wiring decision, not a policy.** NPC-Live gates on its portrait: a player should
see who they are talking to before speaking, and a face arriving mid-sentence is worse than a short
wait. The cost is a pause before the first word rather than a delayed session, since the live
connection opens regardless. A genre that would rather not wait simply leaves that edge out.

**Whether an input is held is read off the wiring too.** The graph counts the release edges leading
into a source and tells it, the same way it tells a gate how many reporters it has, so a source
with no release edge is free from the start — which is what keeps a genre that wired no gate from
waiting forever on one.

A held input keeps what was typed during the wait and delivers it on release, because someone who
typed while a portrait rendered meant to say it. Speech that arrived is dropped instead: replaying
stale audio into a live conversation is worse than losing it.

**The first message is the scenario followed by the cue** — who the character is, then what to do
about it — configured on the gate. v1 calls that cue the initialization prompt, and the name is
kept. The scenario also stands as the session's instruction, so a character holds it whether it
reads the opening message or the instruction it was given.

A genre with nothing to prepare has a gate with no inputs, which opens immediately — so a caller
never has to ask which kind of genre it got.

Making "before play" a real point in time also removes a class of ordering bug: a block acting on a
player's input before the values it needs have arrived, which otherwise resolves correctly most of
the time and not always.

## Session & persistence model

**Auth authorizes launching a session, not playing one.** Once launched, the session id is a
self-sufficient bearer handle — whoever holds it can continue the session. This is what makes
iframe embedding safe: the spendable API key never reaches client-side code, only a scoped session
id does, and that id can't be used to launch anything new. Attaching a session to a user is
optional platform-side bookkeeping (a mapping table the platform keeps outside the engine), not a
structural requirement of a session.

**Two tables, and as few columns as will do the job.** Whatever the engine stores, it stores as a
blob:

| Table | Columns |
|---|---|
| `sessions` | `id`, `schema_version`, `spec` (JSON blob), `state` (JSON blob) |
| `events` | `session_id`, `seq`, `stream`, `value` |

`spec` and `state` are separate because they change at different rates: the spec is fixed at
launch, while `state` is rewritten as the session runs.

**A game's history is the ordered log of everything that reached an output sink.** There is nothing
else to record: whatever the player saw, they saw because it arrived at a sink, so appending each
event as it passes is both the persistence mechanism and the history. That is already how the
engine works — the session folds its sinks into one stream and hands every event to `Persist`.

Two things fall out. The history endpoint is that log replayed in order, and **the client applies a
replayed event through exactly the same code as a live one**, so there is no second rendering path
to keep in step. And a turn needs no bracket in storage: a sequence is enough, and whether the UI
groups events into turns is a display decision rather than a schema one.

Nothing here is a column the platform can query by user, workshop or game — and it doesn't need to
be. Attaching a session to a user is platform-side bookkeeping, so the platform keeps its own
mapping table with whatever it indexes on, and the engine's schema stays this small. That split is
what makes a second `Persist` implementation — SQLite for a standalone build — a morning's work
rather than a schema exercise.

`schema_version` is a column rather than a field inside the blob, so stale sessions can be found
and purged without parsing every document.

The `spec` blob holds no secret by construction: the API key is an injected function, and
`Persist` is too. Both are behaviour rather than configuration, so neither serialises.

Media is keyed by session and turn but stored *alongside* the turn rather than inside it. A
base64'd image in `content` makes every read of that turn expensive, including the text-only replay
path that exists precisely to avoid touching media.

**Schema evolution by invalidation, not migration.** A version mismatch on load means the session
is simply treated as invalid. The marker is explicit rather than "does it unmarshal cleanly" —
Go's JSON decoding is too permissive to catch a semantic change reliably. This is acceptable specifically because sessions are cheap and disposable —
unlike a `Game` (a teacher's authored scenario) or a `User`, nobody loses anything of lasting value
when an old session can't be read after an engine upgrade; they start a new one. Games and Users
keep real migrations; sessions don't need them.

**Adventure fills the event log; NPC-Live barely touches it.** A live conversation is ephemeral by
default: the one thing worth keeping is the portrait made during preparation, and whatever the
observer flagged. The dialogue itself is not stored. That is deliberately unsettled while the genre
is an experiment, and nothing above depends on settling it.

An ephemeral conversation is better for data protection and worse for incident review: nothing
about a young player's dialogue is retained, and equally nothing is available when someone asks what the
character said. The observer's flags are the natural middle ground if one is ever wanted — keep
what was flagged without keeping the conversation.

**Erasing a user touches both sides.** The mapping lives in the platform's table and the content in
the engine's, so deletion is a two-table procedure. It should be written down as one procedure
rather than reconstructed under time pressure.

**Persistence happens at checkpoints during a turn, not once at the end.** This rules out a pure
"state in, state out" function — real intermediate results need saving mid-flight: a generated
image, the plot/prose thread's updated continuation token, a status update. The engine calls a
`Persist`-shaped interface at each of these points, and that interface is **dependency-injected**:
the platform hands in one backed by its Postgres, a standalone build hands in SQLite, a dev
launcher hands in memory or nothing. The graph holds an interface and never learns where anything
lands. This generalizes a pattern that
already exists today — `stream.go`'s `ImageSaver`/`AudioSaver` callbacks already persist media the
moment it's final, with the caller never knowing where it's saved. `Persist` is that same idea,
widened to cover the whole turn.

## Engine/platform code boundary

Two sibling Go modules, joined by `go.work` during development:

- The engine module's internals live almost entirely under its own `internal/` — compiler-
  enforced: `go build` itself refuses an import from outside the module's own tree. Not a
  convention, not a lint rule, not something that depends on review discipline or on a coding
  agent remembering a rule — the code simply won't compile if the boundary is crossed. Checked
  against the scaffold from the `server` module: importing `engine/internal/ports` fails with
  *"use of internal package engine/internal/ports not allowed"*, while importing `engine` builds.
- The engine exposes a small, deliberate public API: a `Launch(spec) (*Session, error)`-shaped
  call, a `NextTurn(session, action) (*Session, *Response, error)`-shaped call, and the `Persist`
  interface described above.
- **The engine knows nothing about HTTP or REST.** No `http.Handler`, no request/response types
  tied to a transport. The portal owns all routing, request parsing, response writing, and SSE
  framing, translating between HTTP and the engine's plain Go calls. This mirrors what already
  exists today: `stream.go` only ever deals in a typed Go channel; `api/routes/sessions_messages.go`
  is the only place that knows SSE exists at all. The split being proposed here is that same
  separation, generalized to the whole engine instead of just the streaming layer.
- Frontend assets (HTML/CSS/compiled TypeScript) are embedded into the engine module via
  `//go:embed` and served from the engine's own subtree, alongside the endpoints they call. That
  is what lets the player address the API with relative URLs and need no configuration: the same
  build works under the platform's mount point and under a standalone server.
- Because the module and `internal/` boundaries are real from day one, moving the engine into its
  own repository later — if that's ever wanted — is a near-zero-cost move: stop workspace-
  including it, tag a version, point at a git URL. Nothing needs untangling first, because nothing
  was allowed to tangle.

### Deployment shapes

**chatgamelab.eu runs the monolith**, and the reason is access control rather than convenience.
In one binary the engine has no network surface of its own, so there is exactly one door: every
request passes the portal's JWT validation, role and membership checks and share-token resolution
before the engine is called at all. A standalone engine would expose its own endpoints, where the
session id — a bearer handle by design — is the only thing left guarding them, because the engine
knows nothing about users and cannot check anything else.

It also keeps a **live gate** in front of a frozen spec. The `SessionSpec` is fixed at launch, but
the portal owns the route, so a revoked key, a changed role or a withdrawn share is enforced on the
next request regardless.

Mechanically: the portal's binary imports the engine and holds sessions as in-memory instances,
with no second process and no wire protocol between them. `Persist` is the portal's Postgres
implementation, passed in at launch.

Because the engine owns no transport, the standalone shape stays available without being built: a
thin `main` wrapping the same calls in HTTP, which is what the dev launcher already is in embryo.
It stays cheap only while three things hold — the engine owns no HTTP, `Persist` stays injected,
and the portal never reaches into `internal/`. The compiler enforces the third.

The monolith's cost used to be that **a portal deploy ended every live conversation**, since a
session is a running graph, its goroutines and an open model connection. Adventure survived it —
turn-based, state in the database — and a voice call did not, which made deploy timing an
operational rule once voice shipped.

The WebRTC topology largely removes that. The conversation runs between the browser and OpenAI, so
restarting the portal drops the engine's sideband rather than the call: the character keeps
speaking, and what stops is our metering, moderation and persistence until something re-attaches.
That is a much smaller failure, and a strictly better one to have chosen — though "the guardrail
observer is offline while the conversation continues" is its own thing to think about, and an
argument for re-attaching automatically rather than treating a deploy as harmless.

The one seam that needs care is `Persist`, because it is an interface the engine *calls*. Injecting
a database connection is ordinary DI whatever the shape. What the engine must never do is call back
into the *platform's API* to save something — that reverses the dependency arrow and hands the
engine a platform URL, credentials and knowledge of platform endpoints. A standalone engine
persists to storage it was given, not to the platform it may know nothing about.

A live session also pins itself to one process: an open model connection plus a running graph and
its goroutines. That is true in both shapes, and only becomes visible if a standalone engine ever
runs more than one replica.

## Usage and cost

Blocks report what they spend, on a port of their own, and the engine aggregates it. For a platform
that teaches how AI works and pays per session, what a turn cost is part of the subject rather than
an operational detail.

**Three units, because providers bill in three ways**: text per token, live conversation per second
of session duration, and pictures per picture. Cached input is counted apart from fresh input
because it is priced apart — the game flow hits the prompt cache from the second turn onwards. A
report carries the *resolved* model name, never a tier reference, since prices attach to models.

The live unit is duration rather than audio: `gpt-live-1` charges for the session being open,
whether anyone is speaking or not, at $0.05 per minute billed per second and never rounded up.
Creating a WebRTC session bills fifteen seconds up front and credits them back once it runs, so a
session abandoned during negotiation still costs something — which is the reason the engine creates
one only when a player actually arrives.

**Totals, not deltas.** A block reports what it has spent so far, so a dropped event costs accuracy
for an instant rather than permanently, and a late subscriber sees the whole session rather than
the part that came after it. The live session needs no ticker to achieve this: GPT-Live sends
`session.usage.updated` as the conversation runs, carrying cumulative seconds that the
documentation warns must not be summed. A running total is precisely the shape this section already
asks for, so the block forwards it rather than counting anything itself.

**The engine prices it, because the price table is configuration.** Providers do not publish prices
through their APIs, so the table is hand-maintained and carries the date it was taken. The report
comes back two ways — per block, for labels on a graph view, and per model, because that is what a
price attaches to — with a flag saying whether every model in the session was priced, so a total is
never a confident-looking floor.

## Protocol

**The session-scoped event stream is the protocol; a turn is a bracketed span within it.** The
engine emits one typed event stream for a session's lifetime, and the client sends input over the
same connection. Adventure's turn loop is then the restricted case — a stream whose events happen
to fall into one bracket per action — rather than a second protocol wearing the same name.

This reverses an earlier draft, which specified REST + SSE and argued against WebSockets. That
argument was sound for its stated premise, traffic that is "turn-based and half-duplex end to end",
and the premise turns out to describe Adventure rather than the engine. NPC-Live is full-duplex by
construction: barge-in alone means audio flows both ways at once. Designing for the duplex case and
letting turn-based fall out of it yields one protocol; the other order yields two that drift.

**A voice genre takes voice.** An earlier draft had typed input riding the same connection as
microphone audio, on the grounds that a live model accepts both — and GPT-Live does not. It has no
event that delivers user text into the conversation: text enters as startup history, as appended
context the model treats as something it knows rather than something it heard, or through a
delegated backend. None of those is a player speaking.

That costs something real, and it should be recorded as a cost rather than smoothed over: someone
without a microphone, or who would rather not speak aloud, cannot play this genre. The honest
response is for the wiring to say so — NPC-Live has no typed-input node — instead of offering a text
box that reaches the character by a route with different meaning. Should that matter more than the
latency does, delegation is the mechanism that gives typed input a genuine home.

**A boundary between utterances travels in-band**, as an empty text delta. The obvious alternative
— a marker on the state stream — cannot work: the two streams reach a reader through different
goroutines, so nothing orders a boundary against the text it is meant to close. In-band, it is
ordered by construction, and it appends nothing, so a reader that does not care may ignore it.

**The wired output blocks define the event schema.** Adventure wires text, image, audio and
status; NPC-Live wires audio, text, image and the observer's flags. The stream's event types are derived from a genre's wiring
rather than being a fixed union each genre partially fills.

**The genre declares its interaction model**, and the client reads that flag to pick its shell.
This is the same capability-driven branching Adventure's Transcribe stage already uses, applied one
level up.

**The live audio topology is settled**, and it is neither of the two an earlier draft weighed
against each other. Those were posed as a choice — browser-direct WebRTC, cheap and fast but
leaving the engine blind; or an engine relay, keeping control at the cost of latency and bandwidth.
GPT-Live offers both at once, through a second connection called a **sideband**:

```
browser ──── WebRTC media + data channel ────> OpenAI
   │                                              │
   └── SDP exchange ──> engine ── sideband WS ────┘
```

The browser holds the audio. The engine creates the session with `POST /v1/live/sessions`, brokering
the browser's SDP offer for an answer, then attaches separately at
`wss://api.openai.com/v1/live/sessions/{id}/attach`. On that sideband it receives both transcripts,
usage, the session's close reason and copies of the audio itself, and it can send every control
event — appended instructions, mute, close. The API key never leaves the engine, and OpenAI's
session id never reaches the browser.

So the objection that ruled WebRTC out is answered rather than accepted: the engine keeps metering,
moderation and persistence while staying off the audio path. What it buys is real. Relayed PCM16 at
24 kHz is about 384 kbit/s each way per session through the portal; WebRTC negotiates Opus at a
fraction of that, none of it ours. Audio over TCP stalls behind a lost packet, where WebRTC conceals
the loss. And a redeploy no longer ends a conversation — see Deployment shapes.

**The graph still draws both audio nodes, and `live-session` requires them.** The wiring describes
the genre's dataflow, not which bytes pass through this process: the live session genuinely does
receive the player's speech and genuinely does send speech to the client, and a diagram that
omitted that would hide the one path the genre is about. The nodes are honest either way — when
reflected audio arrives on the sideband they carry real samples, with nothing in the wiring
changing.

**Connecting is triggered by the wiring, not by a call sequence.** The live block emits a connect
signal once the gate opens; the browser offers SDP only then. So the portrait gates the voice
connection, which is the behaviour this document already argues for — see who you are talking to
before you speak — and it is a wiring decision rather than a rule written into a launcher.

**An idle conversation is let go.** GPT-Live bills for the time a session stays open, whether
anybody is speaking or not, so a player who wanders off costs money until somebody notices. Muting
does not help — the documentation is explicit that muting input leaves the session running. The only
cheaper state is closed.

So after a minute with nothing said on either side, the engine closes the conversation and puts a
`pause` on the stream. The player's one control changes what it shows — a microphone while the
character can hear you, pause bars once the conversation has been let go — and pressing it opens a
new conversation, which the character carries on. The same press the other way lets a conversation
go before its silence does, for a player who knows they are finished. Both directions are a round
trip, so the control holds a transition until the far end says the change happened: a button that
arrived at the far state early would be reporting something that has not occurred yet. What it does
do at once is stop sending: disabling the local track takes effect in the browser, and somebody who
has pressed pause should stop being heard then rather than when the provider agrees. The threshold
is a cost calculation rather than a feel: creating a session bills fifteen seconds up front, so
pausing pays for silences longer than about fifteen seconds and a shorter timeout would cost more
than it saves.

Having let a conversation go, the live block comes straight back round to waiting for an offer and
says so — the same `connect` signal it sent when the gate opened. **Only the first one is acted
on.** A page that answered every invitation would resume a conversation nobody asked to resume, and
start billing for it again, which is the whole of what the pause was for. The later ones are the
engine saying it would take an offer; the player's own press is what sends one.

**A reload takes the same path.** It destroys the peer connection and leaves the provider talking
to nobody, and nothing about that is replayable: the invitation was sent while the new page did not
exist. So the page asks where the conversation stands — `awaiting`, `open` or `paused`, carried on
the snapshot beside the phases and the running cost — and acts on the answer. `open` means a
conversation it has lost, and it offers again; the live block takes a second offer as the player
coming back, closes the conversation they left, and continues it in the successor. That is the
resume path with no pause in front of it, which means the reload case is a live test of the
transcript fallback: the close is graceful, so a fork is tried first, and whatever the provider
kept, the character remembers what was said.

Silence is measured from the last thing **either** speaker said, and never taken while the character
is still talking — a gap between transcript events is not by itself silence, and hanging up
mid-sentence would be worse than the charge.

Resuming has two mechanisms and needs both. A **fork** continues from the recording the provider
kept, which is why sessions are created with `store: true`. But a recording only finalises on a
graceful close, and a dropped connection, a closed laptop or a redeployed portal finalise nothing —
which are exactly the cases somebody most wants to resume from. So the engine also keeps the
conversation in memory and can **seed** a new session with it. That memory holds both halves,
including the player's own words, which are otherwise received and discarded; it is never persisted
and never leaves the process. A character restored with only its own remarks would be answering
nothing.

**A reloading page rebuilds from REST**, which is the same division seen from the client's side:
the socket carries what is happening, and these carry what has happened and where things stand.

| Endpoint | Answers |
|---|---|
| `/topology` | the wiring, which does not change during a session |
| `/state` | where the session stands: phases, status, spend, whether it has started |
| `/history` | the conversation so far, in order |
| `/nodes/{name}` | what one block is and what recently passed through it |
| `/prompts` | the text this session is running on, after any the spec replaced |
| `/edge?from=&to=&kind=` | what one wire has carried |
| `/flowchart` | the same wiring as mermaid |

A replayed event is applied by exactly the same client code as a live one, so there is no second
way to render a session and no second place for it to be wrong. History keeps the conversation and
leaves out audio: replaying thirty frames per utterance into a reloaded page would be neither
useful nor cheap.

**Finished artifacts stay on plain stateless GETs.** Message history and completed media remain
retrievable with no live session or engine involvement at all — enough on its own to review a past
playthrough. This already exists today (`GetMessageStatus`, `GetMessageImage`, `GetMessageAudio`
alongside `GetMessageStream`); the split doesn't change it, it clarifies which half is the engine's
live stream and which half is ordinary retrieval.

## Frontend

**Dependencies are judged one at a time, and the bar is "does it own an event model".** A library
that takes data and paints — tsParticles' vanilla core is the live example — sits underneath the
timeline and has no opinion about it. A framework that wants to own when things render is what
this section rules out. That distinction, rather than a dependency count, is the rule.

### What exists today

A working player lives in `web/`, built with esbuild and TypeScript. `make web` typechecks and
bundles it; `make dev` does that and then serves a session ready to play. Sources are in `web/src`
and the build lands in `web/dist`, which is **committed**: the Go module embeds `dist`, so building
the engine never needs a JavaScript toolchain and only changing the player does.

The layout is split evenly. On the left the conversation — hold-to-talk, a text box, the scene
image, a status bar for genres that track one. On the right the instrumentation: the wiring drawn
as a live graph, and a details panel under it.

**The graph is drawn, not listed.** Cytoscape with a dagre layout, top to bottom so the P reads the
way this document draws it. Sources, blocks, sinks and the gate each have a shape. A block's border
lights while it works and its label carries what it has spent. A finished block keeps an afterglow
for 700 ms, which is the only reason a 40 ms tool call is visible at all — the observer finishes
between two frames, and without it the fastest and most interesting blocks would be the ones nobody
ever sees work.

Cytoscape is the one weighty dependency, at 154 kB gzipped against a 4 kB headless core. It passes
the bar above — it takes data and paints, and has no opinion about the session's timeline — and it
draws a cyclic graph with live per-node styling, which is what the observer loop needs and what a
hand-rolled layout would get wrong. It is confined to `web/src/ui`, so the headless player never
loads it.

**Clicking reads.** A block shows its type, role, state, model, spend and the last few values in
and out, each timestamped and newest first. A wire shows what it last carried — the observer's
steering edge is the one worth clicking, since it shows the guardrail being re-injected. Selecting
either marks it and what it touches. With nothing selected the panel shows the session's spend by
model.

**The player is iframe-embeddable.** A host page sizes the iframe with ordinary CSS — percentage,
flex/grid, `vh`, media queries — and the content inside reflows exactly as if the browser window
itself had resized. No `postMessage` bridge is needed unless content-driven auto-height is wanted
later; an internally-scrolling box is the natural shape for a chat-style feed regardless.

**Built around the live-duplex case, rendered as a chat.** The headless core's timeline is the
superset — a continuous session carrying concurrent streams — and Adventure's turn loop runs on
that same core as the restricted case. The *layout* stays turn-based either way: a live
conversation's transcript is displayed as chat history, so a voice dialogue and a typed adventure
look like the same thing on screen even though only one of them is turn-based underneath.

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

**Genuinely headless, and the layout says so.** The player *is* the headless core: it sits at
`web/src`, and the UI is a subdirectory of it rather than the other way round. The core owns the
whole session state — transcript, flags, status, portrait, phases, spend — with no DOM and no audio
device, so a full game is playable from node. A thin UI layer's only job is to paint whatever the
core's current state says and to pipe input back in.

That is what makes integration testing cheap. A Go test mounts the engine's subtree with `httptest`
and runs the same core from node against it, playing a real session end to end in a tenth of a
second with no browser. Go drives, so there is no port to guess and no readiness to poll, and the
test skips when node is absent. This mirrors the backend split exactly: a library of
narrowly-ported blocks wired per genre on the backend, one headless core with a pure render layer
on the frontend — the same principle, applied on both sides of the boundary.

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
`testutil.suite`. Genre-level correctness (does a status update apply correctly, does a genre's
wiring call blocks in the right order) doesn't need any of that infrastructure at all.

The Live adapter role needs a mock of its own, and it's a harder one: replaying a scripted event
stream, out of order on purpose, is the only way to test that the transcript reconstruction holds
up under what the real API actually does. A scripted stream is also how the observer loop gets
tested without a live model — feed it a drifting, increasingly compliant NPC and assert that
steering fires.

Graph-validity tests per genre are described under Typed ports above.

## Rollout

1. **Scaffold the engine thin, and build NPC-Live's observer loop on it first.** *(Under way — a
   `./engine` module exists with both genres wired from dummy blocks, a graph validator, an
   injected `Persist`, and a dev launcher that runs either genre with no API key and no network.)*
   Leave out the portal, the transport, and any real adapter until the flow is proven. The earlier
   plan here was a throwaway single-file spike outside the engine, on the grounds that building
   gpt-live *as a block* would presuppose the block model fits it. The observer loop changed that:
   it is multi-block by construction, so the thing under test is the wiring itself, and a throwaway
   would mean building that machinery twice in a shape that doesn't transfer. The defence against
   the original worry — sunk cost quietly bending the abstraction — is keeping the scaffold small
   enough to still throw away.
2. **What the spike has to answer**: whether a live model stays in character through a long natural
   conversation, and whether the observer can pull it back when it drifts; which events a session
   actually emits against the documented set; and what a five-minute conversation costs. *(The
   transport question that used to sit here — WebRTC-direct or relayed — is answered above: both,
   through the sideband. The observer runs muted for the first pass, so the voice path is proven
   before a second model is added to it.)*
3. **Adventure** — rebuilt by wiring the building blocks. It's the known-good genre, so it proves
   the block library can express what v1 already does.
4. **NPC-Live** — the forcing function for this whole document. Expected to expose places where the
   block library, shaped by Adventure, cut a seam wrong; that's the plan working as intended rather
   than a sign step 3 was done badly.
5. Once v2 fully works and is wired into the portal, **v1 is purged entirely — frontend and
   backend both.** No prolonged dual-running. Part of the value here is specifically the
   platform-side simplification that comes from deleting the old, entangled implementation, not
   just having a nicer new one sitting next to it.

No further genres are currently planned beyond these two.

## Deliberately deferred

Noted so they aren't silently forgotten, not because they're expected soon:

- **Veto/fact-check block** — a parallel stage with authority to reject a turn's output (e.g. on a
  factual error), sketched into Adventure's diagram above but not designed in detail or built. It
  gets harder rather than easier under NPC-Live: text can be inspected before it renders, whereas a
  live model is already speaking into the player's ear as it generates. For a platform that argues
  its youth protection as carefully as JUGENDSCHUTZ.md does, that gap deserves a decision before
  the voice genre ships, not after.
- **What a live session persists** — ephemeral for now, possibly a single turn 0 holding the
  character's generated image. Settling this also settles whether the event stream needs turn
  brackets at all, which nothing currently requires.
- **Delegation** — GPT-Live can pair with a backend model that works *beside* the conversation
  while the character keeps talking, and explains the result when it lands. This is the largest
  door this document leaves unopened: it is where Adventure's outline-and-expand could run behind a
  voice, where status extraction belongs, where the veto block above would finally have somewhere
  to live, and where typed input regains a genuine route. Two things make it a decision rather than
  an addition — the mode is fixed when a session is created, so it cannot be tried on a running
  one, and a backend bills tokens on top of the voice minutes.
- **Seeding a resumed session with its transcript** — `session.start` accepts up to 128 prior
  messages and 8,192 tokens of history, which is how a dropped or redeployed conversation could
  continue rather than restart. Nothing is stored at OpenAI to make this work; the engine supplies
  what it kept. Not built, and worth having when a live conversation is worth resuming.
- **Per-session spend or duration bound** — pausing an idle session answers the player who walks
  away; it says nothing about the one who talks for an hour. The session id is a bearer handle, so
  every minute taken on it spends the resolved key. Whether a ceiling belongs on the session itself
  is open, and sharpens now that a genre bills by the minute.
- **Locking down the frontend data channel** — under WebRTC the browser can send any GPT-Live
  command on its own data channel, including `session.instructions.append`, which is the event that
  steers the character. `allowed_client_events` and `allowed_server_events` restrict that, and are
  fixed at session creation. Judged a non-problem for now on the grounds that anyone crafting such
  an event has outgrown the guardrail; the residual case is a pasted snippet, which needs no skill.
- **Cadence, and what a hard hit does** — whether the observer consumes transcript deltas as they
  stream or the completed transcript per response (one NPC turn is ten to twenty seconds of
  speech), and whether a guardrail violation lets the current utterance finish or cancels it
  mid-sentence. For this platform the audible cut is probably right, but it's a product call.
- **Per-turn spend bound on a session** — the session id is a bearer handle, and while it can't
  launch anything new, every turn taken on it spends the resolved API key. A voice genre billed per
  minute of audio sharpens this. Whether a turn or time budget belongs on the session itself is
  open.
- **Existing v1 `Game` data at cutover** — whether existing production games need a one-time
  conversion script into the new `SessionSpec` shape, or few enough exist that manual
  re-authoring is simpler, depends on a production game count this document doesn't have visibility
  into.
