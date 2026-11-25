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
opens a *session*. A session is played in the *playback-engine* by a person
called the *player*.

- *Genre* picks one of the engine's ways to play.
- *Scenario* fills in what that genre needs, such as an adventure's background
  story.
- *Image style* describes how the game's pictures look.
- *Voice style* describes how the game's voices sound.
- *Voices*: for each role its genre speaks in, such as a narrator or a
  character, a choice of male, female or any; or custom, where the genre makes
  a voice of its own from the voice style. With male, female or any, the
  session casts a built-in voice that fits the scenario and speaks the
  player's language, and keeps it for the whole session.
- *Theme* picks the look of the playback-engine around the game.

A game has a *cover*: one picture of it, drawn in its image style under the
youth-protection rule for the youngest players, since anyone who sees the game
sees its cover. A cover can also come with a game file; it is checked
against the same rule before it is kept.

A game travels as a *game file*, a `.cgl` file: a zip that holds

- `game.yaml`, the game's fields;
- `cover.jpg`, its cover, when it has one;
- `cover.sig`, the cover's signature from the ChatGameLab that exported it.

== Sessions

A session is one player playing one game. Creating it takes:

- the game;
- the player, who may be a guest;
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
continue it later under that ID, for as long as the session is kept: a
session not played for the site's retention window is deleted. The admin sets
the window, in hours; 0 keeps every session. A finished session can be read
back as it was played.

A session meets *problems*, such as a lost connection; it resolves them by
itself where it can. A session can also be *over*, for instance when the AI
provider's safety filter stops it. That is final: the session stays readable,
and its last line tells the player why.

A session plays on one AI platform at one quality tier (@platforms). The
optional features *images*, *audio-in* and *audio-out* can be switched off or on
at any time during play, if available. *Every player may change the prompts* of
the game they are playing with the exception of the Guardrail prompt, 
which only staff, head and admin may change. Changes apply to all API calls.
Heads and staff may also adapt them before a session starts.

== AI platforms and tiers <platforms>

A session plays on one *AI platform*, such as OpenAI or Mistral, at one
*quality tier*: economy, balanced or high. Together they pick the AI model for
each task in a session:

A tier changes how well a session plays, never which features it has. Which
*features* exist depends on the platform: *images* (pictures), *audio-out*
(spoken narration), *audio-in* (voice input) and *audio-live* (live voice).

The platform shows, for each platform and tier, which AI model runs each task.
The engine comes with a model for each task and tier, and a price for each
model; on a configurable platform the admin assigns other models, from the
platform's model table only, and sets a model's price, until they reset it.
A price the admin set holds until the next release, which brings its own.

== Genres

The genre decides how a game plays. A genre is a game mechanic built into the
engine.

A genre comes as a pair of *halves*: *live*, a conversation by voice as it
happens, and *rounds*, played in turns. Both halves play the same game and only
read it differently, so a game is made once, for its genre. A genre may have a
single half.

- *Each half needs certain features*, and a game plays when the key share used
  for the session provides them for one of its halves.
- *The player picks the half* when the game starts, where both fit; where one
  fits, the game starts straight away. A website that embeds the game may pick
  for the player. Designers and staff limit the halves only through the key
  share.
- *The half stays for the session*: continuing a session continues it in the
  same half.

=== Adventure

The player says or types what their character attempts, and the world reacts.
Adventure plays in rounds.

- *The game keeps control of outcomes.* The player can try anything; what
  happens is decided by a plausible world reacting to the attempt.
- *Each turn* answers with narration, and optionally a picture, spoken
  narration and a change of status.
- *A game can track status* — values the scenario defines, updated as the story
  moves.

=== NPC

The player talks with one character, live or in rounds.

==== NPC-Live

The player talks by voice with one character.

- *The player sees the game first.* The game's picture from the gallery
  appears at once, then the character speaks first: its opening words are
  written from how the game begins, and the conversation opens with them.
- *The character holds its line.* It stays in character through a long
  conversation and keeps to what the scenario says it must not concede. When it
  starts to drift or give way, it is pulled back.
- *The conversation is a transcript* of what the character says; the player's
  own words are heard, and only the character's appear.
- *An idle conversation pauses* once the player has said nothing for a while.
  The player resumes it
  with one press, and the character carries on where it left off. Reloading the
  page carries on too, and so does a dropped connection, by itself.
- *Playing takes a microphone.* The player speaks and the character listens.
- *Playing takes audio-live*, for the conversation; the picture is the game's
  own, so nothing is painted.
- *An NPC-Live session keeps* the game's picture and any warnings raised
  during the conversation; the conversation itself is ephemeral.

==== NPC-Rounds

The player and the character take turns, as in a chat.

- *The player sees the game first.* The game's picture appears, then the
  character speaks first, with opening words written from how the game begins.
- *The player types or speaks a turn*, and the character answers in writing,
  read aloud in the game's voice when audio-out is on.
- *The character holds its line.* The youth-protection rule and what the
  scenario says it must not concede go with its answers.
- *Playing needs no feature of the key*; audio-in and audio-out are optional.
- *An NPC-Rounds session keeps the whole conversation*: it can be continued and
  read back.

=== Discussion

Two characters talk with each other by voice, and the player may join in or
only listen. Discussion plays live.

== Youth protection

Every session plays under exactly one youth-protection rule (Jugendschutz rule),
also called its *guardrail*, on top of the AI provider's own safety measures.
When the session is created, the platform supplies it from a fallback chain of
rules (@jugendschutz), and the session keeps it. *Only staff and heads may change the guardrail*, of their own
session, during play, to experiment with it; a change applies from the next AI
call on.

- *A designer has no influence on the rule.*
- *The rule holds whatever the scenario says.* Scenario and rule stay separate.
- *A live character is held to the rule* as it speaks, and corrected when it
  strays.
- *A safety stop is shown as one.* When the AI provider ends a conversation for
  its content, the player is told so, distinct from a lost connection.

== Insights (Seeing How It Works)<insights>

Transparency is one of ChatGameLab's main goals, as its games are educational.
*Insights* show every player the inner workings of the game they are playing,
alongside the game itself:

- *The steps of a turn* — which AI calls run, in which order, with which
  prompts and results, and which are working right now.
- *What it costs* — a running total while playing, broken down per step and per
  AI model.

== Open questions

- *Rejecting a turn* — a check that can stop a turn's output, e.g. for a factual
  error. A live character is already speaking as it generates, so this needs a
  decision before NPC-Live ships.
- *Cutting a character off* — whether a youth-protection hit stops the character
  mid-sentence or lets it finish.

= The platform

The platform is the website around the games: where people sign in, write games,
run workshops and play.

== Roles

- *Admin* — runs the site: sets the site-wide youth-protection rules per age
  group, the AI quality tiers, the site's free-use key and how long sessions
  are kept, sets how the site
  presents itself (its name, its operator, its support address and the links
  it shows; everyone sees these, also before signing in), chooses the AI model
  for each task and tier of a configurable platform and sets each model's
  price, and upgrades the site to new releases
- *Head* — is a user who *is* part of an organisation, leads it, manages its staff, workshops and rules, runs workshops.
- *Staff* — is a user who *is* part of an organisation, runs workshops within it.
- *Individual* — is a user who is *not* in an organisation, uses the platform on his own.
- *Guest* — plays a shared game through its link, or takes part in a workshop by joining through its link: a guest outside or in a workshop. A Guest can neither enter nor exit a workshop, he's bound to 
the state assigned to him via the share link 

If an organisation invites an individual, that indivudal becomes part of that 
organisation, turning into a head or staff. If a head or staff leaves an organisation 
he becomes an individual again. 
Taking part in a workshop makes no one part of its organisation: guests take
part, and so does an individual through the workshop's invite, keeping their
role.

Anyone who writes a game is its *designer*: they author its scenario, title,
image style, theme and genre.

A game belongs to its designer. The head and staff of an organisation work
together on all of the organisation's games: the games its head and staff
write, and the games written in its workshops. They see, edit, publish, share,
copy and play them as their designers do; a colleague's game outside a
workshop is deleted only by its designer. Whether a game is the
organisation's is read each time: a designer who leaves takes their games
along.

A game shows how often it has been played; every session started counts as a
play.

== Jugendschutz <jugendschutz>

AI providers have their own safety measures. As an educational platform,
ChatGameLab takes responsibility beyond them with a youth protection of its
own: a binding rule that goes with every AI interaction.

=== Schutzstufen

There are rules at three levels:

- *Workshop* — set by a workshop's staff, per workshop.
- *Organisation* — set by an organisation's head or staff, per organisation.
- *Site* — per age group: *U13*, *U13p* and *U18*. Each is the site's
  default below until the admin changes it, and the admin gives it back its
  default at any time; it is never empty.

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

A session's guardrail is chosen when the session is created, by a fallback
chain: the first level in the order below that sets a rule gives the session
its guardrail, and the levels after it do not count. A change of role applies
from the player's next session.

==== Angemeldete Nutzer

1. The shared game's workshop rule, when the game belongs to a workshop.
2. The shared game's organisation rule, when the game belongs to an
   organisation.
3. The current workshop's rule: the workshop the player takes part in. The
   organisation owns it; whoever takes part in it plays under its rule.
4. The organisation's rule: the player's own, for head and staff; otherwise
   the current workshop's organisation.
5. The site rule for the player's age group.

The site rule is never empty, so a signed-in player always has a rule. A
player's own game is not shared, so levels 1 and 2 are empty for it.

- *A shared game brings its protection first.* The organisation that wrote the
  game shaped its content and knows the right frame for it; the player's
  current workshop, organisation and age group follow.
- *An organisation's rule applies whatever the player's age.* The site's age
  groups are mainly for individuals; for a player in an organisation, the site
  rule for their age group stays beneath the organisation's.

#table(
  columns: 2,
  [Role], [Levels],
  [Head, Staff], [shared workshop → shared organisation → current workshop →
    own organisation → site by age],
  [Guest in a workshop], [shared workshop → shared organisation → current
    workshop → its organisation → site by age],
  [Individual], [shared workshop → shared organisation → current workshop →
    its organisation → site by age; outside a workshop, shared workshop →
    shared organisation → site by age],
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

Insights (@insights) show the guardrail the session plays under and where it
came from: a workshop, an organisation, or the site rule for U13, U13p or
U18.

=== Registration

The first sign-in creates an account; the account is usable once registration
is complete. Until then the account is in sign-up and holds no role:
registration gives it its role, individual or the head of the organisation it
registers. Every registered account holds a role. An account left in sign-up
for a day is removed; signing in again starts a new one. Registration asks for:

1. *Nutzertyp.* "Anmeldung als Nutzer" leads on to the Altersabfrage.
   "Als Organisation registrieren" leads on to the organisation's
   registration (below).
2. *Altersabfrage*, by Selbstauskunft:
   - under 13: no account of their own; ChatGameLab only through a
     Jugendeinrichtung;
   - 13–17 without parental consent: U13, the stricter rule;
   - 13–17 with parental consent: U13p;
   - 18 or over: U18.
3. *Name*, required.
4. *Email*, optional.

*Age is a Selbstauskunft.* It is the legally sound way: a technical check, such
as owning an API key, does not hold up legally, and whoever gives a false age
puts themselves in the wrong.

=== Registration as Organisations

A new organisation can be registered by signing up a new user in the 
"as organisation" flow. The new user fills out all the required details 
and ends up as the head of the newly founded organisation.

- *The person* is an adult (U18), reached at their account's email. They give
  their name and function, and agree to the terms of use and the privacy
  policy.
- He give the *organisations* name, its Träger and its address, and sets its
  youth-protection rule.
- *The Einverständniserklärung* binds the organisation, in German, in a
  numbered version. The person agrees to it in the organisation's name; the
  consent is kept as proof.

== Paying

Whoever pays for a session — a workshop, an organisation or the player — is
resolved by the platform. Whoever shares a key decides how it may be used: which
tiers, which features at each tier, a default, and whether players may choose
within that. Several shared keys can be offered at once, and the player picks
one. A revoked key stops working on the next turn.

Drawing a game's cover is paid like a session of that game, for whoever has it
drawn. A voice preview and a review of the form are paid like a session, for
whoever asks for them, but never by a game's sponsor: a sponsor pays only for
playing.

*An API-key share is like a credit-line, it allows it's holder to spend money.* 
It belongs to the user who owns the API-key it, and only they share it
or take it back. If user is member of an organisation it can shared with that organisation. 
All members of that organisation can then use the key within that organisation 
and its workshops. 

== Notifications

A bell beside the avatar tells a user about important facts that are true
now and need their acknowledgement. It shows what is true at the moment: an
entry goes once the user acknowledges it or once it stops being true, and
the next new fact starts a fresh entry. Facts of one kind fold into one
entry with their number. Each user acknowledges for themselves; a user hears
of facts from when they got their role on.

- *Admins* hear of new organisations, and of verified pages waiting for
  their approval; that count drops by itself as pages are decided or
  removed.
- *Head and staff* hear that an admin approved their organisation's verified
  page, or turned it down or withdrew the approval. Only an admin's decision
  counts: a change of site by the organisation itself stays silent.

An entry leads to where the user acts on it. The bell is for signed-in
users with a role of their own; guests play without it.

== Data

Erasing a user removes their sessions along with their account.

== Open questions

- *Spend limits* — a ceiling on what one session can cost.
- *A measure of engagement* — such as turns played next to plays, so games
  people stay with stand out.

= Embedding

A game can run on another website, with no platform around it, as part of
another project. The platform then serves as the editor: designers write and
edit the game there, and embed it on the host website. Players visit only the
host website.

- *An embedded game is one game*, framed by the host page and sized by it.
- *Players need no account*: the host page is the way in. They play as guests.
- *A changed game reaches the host by itself*: the frame always plays the game
  as it is now.

== The embed link

Only verified pages can embed a game. Admins maintain that allow list. 
A game has one embed link, switched on or off; the host page holds it in its snippet. 
Switching it off stops the
snippet; switching it on again brings back the same snippet. 

The embed is configured to a shared API keys whose share pays for its sessions,
at its tier, with the link's play limit and the key's caveats, like a share
link. Changing the paying key keeps the snippet.

== Verified pages

A verified page belongs to an organisation; its head and staff register and
manage it, and an admin needs to approve it. Only one verified page per organisation.
It is defined by a page address and a callback address on the same site.
The admin approves the site, its scheme and host: any port and path on it may
show the organisation's games.

Before a game starts on a verified page, the platform asks the page's callback,
naming the page ID, the host's ID of its visitor and an ID of the request. Only
a yes within a few seconds starts the game.

Every start on a verified page is a new game. One visitor ID of the host is
one guest per embed link: a returning visitor plays as the same guest, another
visitor as another guest. That guest exists only for embedded play: the host
page is its only way in.

