#import "template.typ": spec

#show: spec.with(title: "ChatGameLab")

ChatGameLab is a platform where young people play and make AI-driven games,
and learn how AI works by watching it work. This spec describes what the
platform does, as seen by the people using it.

#block(
  width: 100%, inset: 10pt, radius: 3pt, stroke: 0.6pt + luma(160),
  fill: luma(248),
)[
  *About this spec.* It is the source of truth for what ChatGameLab does: a
  feature is specified here before it is built.

  - *Behaviour only.* Each feature is described precisely, by what a user can
    do and what they observe in return.
  - *Implementation agnostic.* The system is a black box: no code, components,
    technologies, data structures or interfaces between parts.
  - *Human-controlled.* The spec stays short; what goes into it is a
    maintainer's decision.
]

= Roles

- *Admin* — runs the site: sets the site-wide youth-protection rules per age
  group.
- *Head* — leads an organisation: manages its staff, workshops and rules.
- *Staff* — runs workshops within an organisation.
- *Participant* — takes part in a workshop, including by joining through a link.
- *Individual* — uses the platform without an organisation.
- *Guest* — plays a shared game without an account.

Anyone who writes a game is its *designer*: they author its scenario, title and
image style.

= Playing a game

Starting a game opens a *session*. The session's link is all a player needs to
continue it, and a player can leave and come back to where they were. A
finished session can be read back as it was played.

Every game belongs to a *genre*, which decides how it plays.

== Adventure

The player says or types what their character attempts, and the world reacts.

- *The game keeps control of outcomes.* The player can try anything; what
  happens is decided by a plausible world reacting to the attempt.
- *Each turn* answers with narration, and optionally a picture, spoken
  narration and a change of status.
- *A game can track status* — values the designer chose, updated as the story
  moves.

== NPC-Live

The player talks by voice with one character.

- *The player sees the character first.* A portrait appears, then the character
  greets the player and the conversation opens.
- *The character holds its line.* It stays in character through a long
  conversation and keeps to what the designer said it must not concede. When
  it starts to drift or give way, it is pulled back.
- *The conversation is a transcript* of what the character says; the player's
  own words are heard, and only the character's appear.
- *An idle conversation pauses* after a minute of silence. The player resumes it
  with one press, and the character carries on where it left off. Reloading the
  page carries on too.
- *Playing takes a microphone.* The player speaks and the character listens.

= Youth protection

Every AI interaction carries exactly one youth-protection rule, on top of the AI
provider's own safety measures.

- *The rule is resolved live*, per turn: the first rule set among the shared
  game's workshop, the shared game's organisation, the player's own workshop,
  the player's own organisation, and the site's rule for the player's age group.
  Guests follow their own cascade.
- *Only the platform sets it.* The scenario and the rule stay separate, so the
  rule holds whatever a designer writes.
- *A live character is held to the rule* as it speaks, and corrected when it
  strays.
- *A safety stop is shown as one.* When the AI provider ends a conversation for
  its content, the player is told so, distinct from a lost connection.

= Seeing how it works

ChatGameLab teaches how AI works, so a game can show its own workings:

- *The steps of a turn* — which AI calls run, in which order, and which are
  working right now.
- *The prompts* a session runs on, including the genre's own instructions.
- *What it costs* — a running total while playing, broken down per step and per
  AI model.

= Quality and cost

A session runs at one *quality tier* — economy, balanced, premium or max — which
sets the AI models used and the size and quality of its pictures. Economy is a
real rung: a workshop is a room of young people on a school's budget. Some
features, such as spoken narration, are reserved for higher tiers.

Whoever pays for a session — a workshop, an organisation or the player — is
resolved by the platform. A key that is revoked stops working on the next turn.

= Embedding

A single game can be embedded in any website as an iframe, sized by the host
page. The platform shows its own games the same way.

= Data

- *Adventure sessions are kept*: everything the player saw, in order.
- *Live conversations are ephemeral*: a session keeps the character's portrait
  and any warnings raised during the conversation.
- *Erasing a user* removes their sessions along with their account.

= Open questions

- *Rejecting a turn* — a check that can stop a turn's output, e.g. for a factual
  error. A live character is already speaking as it generates, so this needs a
  decision before NPC-Live ships.
- *Spend limits* — a ceiling on what one session can cost.
- *Typing to a live character* — a route for players who cannot or would rather
  not speak.
- *Cutting a character off* — whether a youth-protection hit stops the character
  mid-sentence or lets it finish.
- *Existing games at the switch to the new engine* — converted or re-authored.
