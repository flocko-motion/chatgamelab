#import "template.typ": spec, todo

#show: spec.with(title: "ChatGameLab")

#block(
  width: 100%, inset: 10pt, radius: 3pt, stroke: 0.6pt + luma(160),
  fill: luma(248),
)[
  *About this spec.* It is the source of truth for what ChatGameLab does: a
  feature is specified here before it is built.

  - *Target state.* The spec describes ChatGameLab as it is meant to be. Where
    the running system differs, the difference is tracked as a GitHub issue.
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
provider's own safety measures. When the session is created, the platform
supplies the stack of rules that apply (@jugendschutz); the stack, together
with the platform's own guidance, is condensed into one concise rule, which the
session keeps for its lifetime.

- *A designer has no influence on the rule.*
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

== Jugendschutz <jugendschutz>

AI providers have their own safety measures. As an educational platform,
ChatGameLab takes responsibility beyond them with a youth protection of its
own: a binding rule that goes with every AI interaction.

=== Schutzstufen

There are rules at three levels:

- *Workshop* — set by a workshop's staff, per workshop.
- *Organisation* — set by an organisation's head, per organisation.
- *Site* — set by the admin, per age group: *U13*, *U13p* and *U18*. It is set
  once when the site is installed and is never empty.

The site's defaults:

- *U13*, 13–17 without parental consent: "Erlaube spürbare Konflikte und
  stärkere Spannung; greife bei realistischer, detaillierter oder eskalierender
  Gewalt ein und verhindere Verherrlichung und Überforderung."
- *U13p*, 13–17 with parental consent, and *U18*, 18 and over: "Erlaube
  realistische, auch deutlich intensivere Inhalte; unterbinde exzessive oder
  verherrlichte Gewalt sowie extreme, entwürdigende oder diskriminierende
  Darstellungen konsequent." The two are the same by default; the admin can
  set them apart.
- *Under 13*: the strictest rule. Children under 13 cannot have an account of
  their own; they use ChatGameLab only through a Jugendeinrichtung.

=== Which rules apply

A session's rule is composed when the session is created, from every level
that applies, in the order below. The order is the priority when the stack is
condensed. A change of role applies from the player's next session.

==== Angemeldete Nutzer

1. The shared game's workshop rule, when the game belongs to a workshop.
2. The shared game's organisation rule, when the game belongs to an
   organisation.
3. The player's own workshop rule, when the player is in a workshop.
4. The player's own organisation rule, when the player belongs to one.
5. The site rule for the player's age group.

The site rule is never empty, so a signed-in player always has a rule. A
player's own game is not shared, so levels 1 and 2 are empty for it.

- *A shared game brings its protection first.* The organisation that wrote the
  game shaped its content and knows the right frame for it; the player's own
  workshop, organisation and age group follow.
- *An organisation's rule applies whatever the player's age.* The site's age
  groups are mainly for individuals; for a player in an organisation, the site
  rule for their age group stays beneath the organisation's.

#table(
  columns: 2,
  [Role], [Levels],
  [Head, Staff], [shared workshop → shared organisation → own workshop → own
    organisation → site by age],
  [Participant, also when joined by link], [shared workshop → shared
    organisation → own workshop → own organisation → site by age],
  [Individual], [shared workshop → shared organisation → site by age],
)

==== Gäste

A guest plays a shared game without an account:

1. The shared game's workshop rule, when the game belongs to a workshop.
2. The shared game's organisation rule, when the game belongs to an
   organisation.
3. The rules of the account that created the share, by the levels for a
   signed-in player: its workshop, its organisation and the site rule for its
   age group.

The guest is unknown, so the protection the game's publisher chose for
themselves stands in for theirs; as level 3 ends in a site rule, a guest always
has a rule. A signed-in player opening a shared link follows the levels for
signed-in players.

=== Transparency

The view of a game's workings shows which rules went into the session's rule,
where each came from — a workshop, an organisation, or the site rule for U13,
U13p or U18 — and the rule the session plays under.

=== Registration

1. *Nutzertyp.* "Anmeldung als Nutzer" leads on to the Altersabfrage.
   "Anmeldung als Fachkraft / Organisation" explains how an organisation
   registers, by email, and asks the person to register as a user first.
2. *Altersabfrage*, by Selbstauskunft:
   - under 13: no account of their own; ChatGameLab only through a
     Jugendeinrichtung;
   - 13–17 without parental consent: U13, the stricter rule;
   - 13–17 with parental consent: U13p;
   - 18 or over: U18.

*Age is a Selbstauskunft.* It is the legally sound way: a technical check, such
as owning an API key, does not hold up legally, and whoever gives a false age
puts themselves in the wrong.

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
