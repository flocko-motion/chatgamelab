#import "template.typ": spec, todo

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

== Games

A *game* is a title, a genre, a scenario, an image style and a theme. Playing it
opens a *session*.

- *Genre* picks one of the engine's ways to play.
- *Scenario* fills in what that genre needs, such as an adventure's background
  story.
- *Image style* describes how the game's pictures look.
- *Theme* picks the look of the player around the game.

== Sessions

A session is one player playing one game. Creating it takes:

- the game;
- the player, who may be an anonymous guest;
- one or more shared API keys;
- optionally, the player's preferences.

Before the game begins, the player chooses where there is something to choose:

- *One key, fixed*: the game starts straight away with that key's settings.
- *One key, customizable*: the player tunes the settings within what the key
  allows, such as picking one of its quality tiers.
- *Several keys*: the player picks one, then tunes it if it is customizable.

A session runs in the player's preferred language: the game is translated into
it when the session is created.

Each session has its own ID. A player can leave a session at any time and
continue it later under that ID, for as long as the session is kept.
#todo[How long a session is kept.] A finished session can be read back as it
was played.

A session plays on one *AI platform* (e.g. OpenAI) at one *quality tier*:
economy, balanced or high. A tier presets the AI models a session uses,
including the size and quality of its pictures. Which *features* exist depends
on the platform: *images* (pictures), *audio-out* (spoken narration), *audio-in*
(voice input) and *audio-live* (live voice).

The optional features *images*, *audio-in* and *audio-out* can be switched off
or on at any time during play, as far as the key in use allows the feature.
Switching one off affects only what is made from then on.

== Genres

The genre decides how a game plays. A genre is a game mechanic built into the
engine.

Each genre needs certain features, and a game plays only when the key share used
for the session provides them.

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
- *Playing takes audio-live and images*, for the conversation and the portrait.
- *An NPC-Live session keeps* the character's portrait and any warnings raised
  during the conversation; the conversation itself is ephemeral.

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

Transparency is one of ChatGameLab's main goals, as its games are educational.
The player has a view showing the inner workings of the game:

- *The steps of a turn* — which AI calls run, in which order, with which
  prompts and results, and which are working right now.
- *What it costs* — a running total while playing, broken down per step and per
  AI model.

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
image style, theme and genre.

A game shows how often it has been played; every session started counts as a
play.

== Youth protection

The platform supplies each session's rule, resolved live per turn: the first
rule set among the shared game's workshop, the shared game's organisation, the
player's own workshop, the player's own organisation, and the site's rule for
the player's age group. Guests follow their own cascade. A designer has no
influence on the rule.

== Paying

Whoever pays for a session — a workshop, an organisation or the player — is
resolved by the platform. Whoever shares a key decides how it may be used: which
tiers, which features at each tier, a default, and whether players may choose
within that. Several shared keys can be offered at once, and the player picks
one. A revoked key stops working on the next turn.

== Data

Erasing a user removes their sessions along with their account.

== Open questions

- *Spend limits* — a ceiling on what one session can cost.
- *A measure of engagement* — such as turns played next to plays, so games
  people stay with stand out.

= Embedding

A game can run on another website, with no platform around it. The platform
then serves as the admin interface: designers write and edit the game there, and
export it for embedding. Players visit only the host website.

- *An embedded game is one game*, framed by the host page and sized by it.
- *Players need no account*: the host page is the way in.

== Open questions

- *What an export contains*, and how a changed game reaches the host website.
- *Who pays* for an embedded game's sessions, and which keys it offers.
- *Which youth-protection rule* an embedded game plays under.
