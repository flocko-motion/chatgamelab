#import "template.typ": spec

#show: spec.with(title: "ChatGameLab")

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

= What ChatGameLab is

ChatGameLab makes AI-driven games that young people play, write, and learn from:
a game shows its own workings, so playing it teaches how AI works. Games run on
the ChatGameLab platform, or embedded in another website on their own.

= The game engine

A *game* is a scenario, a title, an image style and a genre. Playing it opens a
*session*.

== Sessions

A session's link is all a player needs to continue it; a player can leave and
come back to where they were. A finished session can be read back as it was
played.

== Genres

The genre decides how a game plays.

=== Adventure

The player says or types what their character attempts, and the world reacts.

- *The game keeps control of outcomes.* The player can try anything; what
  happens is decided by a plausible world reacting to the attempt.
- *Each turn* answers with narration, and optionally a picture, spoken
  narration and a change of status.
- *A game can track status* — values the scenario defines, updated as the story
  moves.

=== NPC-Live

The player talks by voice with one character.

- *The player sees the character first.* A portrait appears, then the character
  greets the player and the conversation opens.
- *The character holds its line.* It stays in character through a long
  conversation and keeps to what the scenario says it must not concede. When it
  starts to drift or give way, it is pulled back.
- *The conversation is a transcript* of what the character says; the player's
  own words are heard, and only the character's appear.
- *An idle conversation pauses* after a minute of silence. The player resumes it
  with one press, and the character carries on where it left off. Reloading the
  page carries on too.
- *Playing takes a microphone.* The player speaks and the character listens.

== Youth protection

Every session plays under exactly one youth-protection rule, on top of the AI
provider's own safety measures. The rule comes with the session, from whoever
starts it.

- *The rule holds whatever the scenario says.* Scenario and rule stay separate.
- *A live character is held to the rule* as it speaks, and corrected when it
  strays.
- *A safety stop is shown as one.* When the AI provider ends a conversation for
  its content, the player is told so, distinct from a lost connection.

== Seeing how it works

A game can show its own workings:

- *The steps of a turn* — which AI calls run, in which order, and which are
  working right now.
- *The prompts* a session runs on, including the genre's own instructions.
- *What it costs* — a running total while playing, broken down per step and per
  AI model.

== Quality

A session runs at one *quality tier* — economy, balanced, premium or max — which
sets the AI models used and the size and quality of its pictures. Some features,
such as spoken narration, are reserved for higher tiers. 

NOTE: A single scalar for quality might be to coarse. We could reduce the tiers 
to just three and in turn introduce toggles like audio-out on/off, images on/off. 
For an api key shared with a workshop of used for a public game, the sharer would
then configure either a fixed setting or a limit for settings (e.g. max tier, 
allowed toggles)

== What a session keeps

- *An Adventure session keeps* everything the player saw, in order.
- *An NPC-Live session keeps* the character's portrait and any warnings raised
  during the conversation; the conversation itself is ephemeral.

== Open questions

- *Rejecting a turn* — a check that can stop a turn's output, e.g. for a factual
  error. A live character is already speaking as it generates, so this needs a
  decision before NPC-Live ships.
- *Typing to a live character* — a route for players who cannot or would rather
  not speak.
- *Cutting a character off* — whether a youth-protection hit stops the character
  mid-sentence or lets it finish.

= The platform

The platform is the website around the games: where people sign in, write games,
run workshops and play.

== Roles

- *Admin* — runs the site: sets the site-wide youth-protection rules per age
  group.
- *Head* — leads an organisation: manages its staff, workshops and rules.
- *Staff* — runs workshops within an organisation.
- *Participant* — takes part in a workshop, including by joining through a link.
- *Individual* — uses the platform without an organisation.
- *Guest* — plays a shared game without an account.

Anyone who writes a game is its *designer*: they author its scenario, title,
image style and genre.

== Youth protection

The platform supplies each session's rule, resolved live per turn: the first
rule set among the shared game's workshop, the shared game's organisation, the
player's own workshop, the player's own organisation, and the site's rule for
the player's age group. Guests follow their own cascade. A designer has no
influence on the rule.

== Paying

Whoever pays for a session — a workshop, an organisation or the player — is
resolved by the platform, along with the session's quality tier. A revoked key
stops working on the next turn.

== Data

Erasing a user removes their sessions along with their account.

== Open questions

- *Spend limits* — a ceiling on what one session can cost.
- *Existing games at the switch to the new engine* — converted or re-authored.

= Embedding

A game can run on another website, with no platform around it. The platform
then serves as the admin interface: designers write and edit the game there, and
export it for embedding. Players visit only the host website.

- *An embedded game is one game*, framed by the host page and sized by it.
- *Players need no account*: the host page is the way in.

== Open questions

- *What an export contains*, and how a changed game reaches the host website.
- *Who pays* for an embedded game's sessions, and at which quality tier.
- *Which youth-protection rule* an embedded game plays under.
