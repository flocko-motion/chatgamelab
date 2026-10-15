# Workshop links: status and follow-up

This file covers the concept 'Workshop-Links merkbar machen + öffentliche
Workshop-Seite'. Branch `feat/memorable-workshop-links` (PR #306) built Part A;
branch `feat/public-workshop-page` builds Parts B and C. For each part the file
records what exists, where it departs from the concept, and what remains.

## Part A: built

- **Word tokens.** Invite links carry three German words, re-login links four
  (`server/functional/wordtoken`). The list is dys2p `de-2048-v1` (CC0) with 34
  words unfit for schools replaced; `server/functional/wordtoken/README.md` names
  them. Old `ws-…` and `participant-…` tokens keep working, and existing
  participants keep their long tokens until someone resets their access.
- **Tolerant input.** Case, spaces, dots, underscores and umlauts are
  normalised on the server and in the browser; `/code` takes three words
  (invite) or four (re-login). The landing page links to it.
- **Reset access.** `POST /api/workshops/participants/{id}/token/reset`, with the
  permissions of reading the token. The re-login popup offers it behind a
  confirmation.
- **Own code in the profile.** Participants open 'Profil' from the avatar menu
  or the phone menu. The profile starts with their access code: the words, a
  QR code and the `/code` link, laid out for one photo
  (`GET /api/users/me/participant-code`). A participant with an old long token
  sees QR code and link, and a hint that the workshop leader can issue a short
  code through 'Zugang zurücksetzen'. Participants never see the age group
  switch, because it would let them loosen their own youth protection.
- **Dead invites.** Asking for the invite link of a workshop whose pending
  invite has expired or is used up now marks that invite expired and creates a
  fresh one (`db.CreateWorkshopInvite`).
- **Token lock.** Lookups of invite, participant and play tokens that were never
  issued count as guesses (`server/tokenlock`). More than 300 within ten minutes
  lock every token check, correct ones included, for five minutes: `429`,
  `Retry-After`, code `token_locked`. `TOKEN_LOCK_MAX_FAILURES`,
  `TOKEN_LOCK_WINDOW` and `TOKEN_LOCK_DURATION` override the defaults. The backend
  drops a session cookie whose token no longer exists.
- **Retired codes do count.** An earlier version of this file claimed tokens of
  deleted users, deleted workshops or reset access never count. They do.
  `ParticipantTokenExists` compares against the *current* column value, a reset
  overwrites it, and `db.DeleteUser` removes the row outright, so
  `db.ParticipantTokenKnown` is false for a code that was once real and
  `httpx.Authenticate` counts it. Only *invite* tokens behave as that sentence
  described, because the used-up row is kept with status `expired` and
  `InviteTokenExists` is status-blind. In a classroom this costs a few failures
  per stale device, not a stream: the backend clears a session cookie carrying a
  retired token, and the frontend clears localStorage on failure. Reaching 300 in
  ten minutes takes a script, which is the point of the lock.
- **Deliberate non-goal.** The answer must never reveal whether a code once
  existed. Distinguishing "reset" from "never issued" would turn the lock into an
  oracle for a 44-bit code. That is why the text on `/code` lists what may have
  happened instead of naming it.
- **Share view.** One full-screen component
  (`web/src/common/components/share/FullscreenQrOverlay.tsx`) for invite,
  re-login and public-page links: a coloured title bar, the QR code, always
  black on white, and the copyable link. See 'Parts B and C: built' for the
  short links it shows. Heads and staff get a QR button next to 'Organisator'
  in the workshop header.

## Part A: departures from the concept

- The lock counts failures site-wide instead of per IP, as decided in review.
  Everyone in a classroom shares one IP behind the school's NAT, and the server
  never sees client IPs (nginx forwards them, the Go code does not read them).
  No `TRUST_PROXY` flag was needed.
- The participant token is the session. During a lock, participants who are
  already logged in therefore get `429` on every request until it lifts. The
  frontend shows 'Zu viele Fehlversuche – bitte später erneut versuchen' and
  keeps the stored token.
- 'Zugang zurücksetzen' sits inside the re-login popup, one click from the
  existing link button, which keeps each participant row at its current icons.
- The concept placed the limit on `POST /api/auth/participant-login`. The web
  app never calls that endpoint for re-login links: it stores the token and
  sends it as a bearer token on every request. The count therefore lives in the
  auth middleware (`server/api/httpx/auth.go`).
- Deleting a workshop already deletes its participant accounts
  (`db.DeleteWorkshop`), so no change was needed there.

## Part A: before the first real workshop

1. Have the word list read aloud and typed in once, as the concept asks. The
   replacement pool is dys2p `de-7776-v1`, same licence;
   `wordtoken_test.go` enforces the format rules on every change.
2. Scan a QR code in light and dark mode, and the full-screen code from about
   three metres.
3. Open an old `ws-…` invite and an old `participant-…` re-login link taken from
   the database.
4. Walk through the new screens in a browser: invite popup in both settings
   views and in the workshop header, re-login popup with reset, `/code`, and the
   landing-page button. The branch passed type checks, lint on the changed
   files, the build, and the Go unit and integration suites, but nobody has
   looked at the screens yet.
5. Done on `feat/participant-profile-code`: `./run-translate.sh` translated the
   new texts into the other 35 languages and brought `server/lang/locales` back
   in step with the web copy.

## Part E: not locking participants out

- **The accept call carries the caller's credential.** `handleAccept` in
  `web/src/routes/invites/$token.accept.tsx` deleted the session cookie from
  JavaScript, which cannot work on an HttpOnly cookie, and then posted without an
  Authorization header. A participant the backend could not recognise was treated
  as a newcomer: a second anonymous account, and `storeParticipantToken`
  overwriting the code of the first. Measured in the dev environment: same person,
  cookie cleared, account `a40de2ff` became `3b28cec7` and the code changed. The
  first account keeps its membership and games, and its old code still works - so
  whoever photographed it can return, which is luck, not design. It now sends the
  token like its sibling `handleEnterWorkshop`, and two answers that only a
  recognised caller can get are handled: 409 means "you are already in" and
  navigates there, and anything from 500 up falls back to the translated text
  instead of showing a raw English server sentence.
- **A reset with nothing to reset fails.** `UpdateParticipantToken` keeps its
  `AND participant_token IS NOT NULL` guard on purpose - a registered participant
  has a real login, and minting a word code for them would add a second, weaker
  credential - but the query was `:exec`, so zero rows looked like success and the
  endpoint answered 200 with a code it had never stored. It is `:execrows` now and
  answers 404, the same as the read side. Reachable for anyone who joined with
  their own account, and for an admin naming any user id, since the admin branch
  returns before the target is loaded.
- **The inactive screen keeps the code.** Its only action used to be a logout that
  erases the stored code, on a screen where the code cannot be fetched back: the
  middleware answers 403 before any handler runs, and `/auth/participant-login`
  refuses while the workshop is off. It now offers the code first, read straight
  from localStorage with no request at all, and warns before the logout. Without a
  stored code it says the leader can look it up, which is true - the leader-side
  read does not check `workshop.active`. The cookie-only participant still depends
  on the leader; letting the code endpoint through the middleware for
  `workshop_inactive` would cover them and is the open follow-up.
- **The screen was English.** `common.workshop.inactive.title` and `.description`
  existed in neither locale, so German participants read the component's inline
  fallbacks.
- **Removing a participant says what it does.** It hard-deletes the account and
  every game they created; the dialogue promised only removal from the workshop.
  The behaviour is unchanged - see 'Bigger rebuilds' - but the warning is honest
  now, and distinguishes the visiting individual, who keeps both.
- **No modal at app start.** `showErrorModal` called while the app is still
  booting sets its store but never paints. Measured while building the above; it
  affects any error raised that early, not just participant auth. Worth fixing at
  the source rather than working around it per call site.

## Bigger rebuilds, for the week of 2026-11-20

Found while hardening the above. All pre-existing, all larger than a fix:

- `DELETE /api/users/{id}` hard-deletes a participant **and every game they
  created**. A leader can erase a registered user's games this way.
- `db.DeleteWorkshop` deletes every participant account and their games, while
  explicitly sparing the games of non-participant members.
- `db.AcceptOpenInvite` deletes **all** of a user's roles, not just the one for the
  workshop being left. Afterwards `getGamesVisibleToUser` filters the games of the
  former workshop out - for their own creator. The data is there, unreachable.
- `db.RemoveMemberFromWorkshop` accepts real participants and removes only the
  role, leaving an account whose token can never authenticate again.
- `PATCH /api/workshops/{id}` replaces rather than merges, and `active` is a plain
  bool: a caller that omits it deactivates the workshop and locks out the cohort.
- The single-role invariant is the root of three of these. Loosening it is the
  actual rebuild.

## Open decisions

- **Scaling the backend.** The lock lives in memory, which suits the single
  backend container in `docker-compose.yml`. More than one instance would need
  the counter in Postgres or a shared cache. The same holds for the mutex that
  serialises the public page's share links (`server/db/workshop_public.go`).

## Parts B and C: built

- **Publishing workshop games.** Whoever may edit a workshop game may switch
  `public` on: its creator, admins, and heads and staff of the workshop's
  institution (`db.UpdateGame`). A game outside a workshop stays with the old
  rule: only its creator. The switch acts at once, because the leader
  decides what the participants want.
- **The page.** `/w/<slug>` exists while `workshop.public` is on, which is off
  by default. It shows the workshop's name, its description (plain text with
  line breaks, up to 2000 characters), and the workshop's public games with name
  and description. It names no creator and no institution. Paused and inactive
  workshops keep their page. A slug that was never issued, was renamed, belongs
  to a switched-off page or to a deleted workshop gives the same 404, and the
  page shows one friendly message for all four.
- **The slug.** Migration 033 adds `workshop.public_slug` (unique) and
  `public_description`. A new workshop gets `<name>-<word>-<word>`: the name
  transliterated, lowercase, cut at a word boundary to 30 characters. Workshops
  created before the migration get theirs from `db.BackfillWorkshopPublicSlugs`
  when the backend starts, because the word list lives in Go. Leaders may edit
  the slug (`a-z`, `0-9`, single hyphens, 3 to 60 characters, unique across the
  site); the old slug then stops working, which the settings warn about before
  saving. Deleted workshops keep their slug, so an old link never leads to a
  different workshop.
- **Settings.** Both workshop settings views carry an 'Öffentliche Seite'
  section: the on/off switch, the slug editor, the description, and the link
  with its QR code in the full-screen view. Participants cannot change it.
  While the page is on, heads and staff find a second QR button for it next to
  'Organisator' in the workshop header.
- **Short links.** Invite and re-login views show `/code/<words>`: three words
  lead to the invite, four log the participant in as `/code` does
  (`web/src/routes/code/$code.tsx`). Any other count opens the `/code` form
  prefilled, with its error. Old `ws-…` and `participant-…` tokens keep their
  long links.
- **Full-screen views.** Invite and re-login links open straight in the
  full-screen view with a blue title bar and the buttons 'Einladungslink
  widerrufen' or 'Zugang zurücksetzen'; the public page's link has a green bar.
  Each view closes through 'Schließen' or Esc only, so a click on the projector
  screen leaves it open.
- **Header.** Visitors who are not logged in see the guest header of guest
  play on a light page, with the footer on desktop. Logged-in visitors keep the
  normal layout.
- **Download.** `GET /api/public/workshops/{slug}/games/{id}/yaml` serves the
  existing YAML export without login, for public games on a page that is on.
- **Copy.** Logged-in visitors copy through the same prefilled create dialogue
  as 'All games'. Visitors without an account go through login, and registration
  if needed, and return to the page (`web/src/common/lib/returnTo.ts`). A second
  sessionStorage entry names the game; the page reads and deletes it once and
  opens the prefilled dialogue. Saving opens the new game's editor.
- **Play.** Each playable public game gets a share link of its own, created on
  the first page visit, paid by the workshop's key and limited to 50 sessions.
  The card shows the sessions left. Without a workshop key every card shows a
  disabled 'Spielen' with a note that no API key is set; the page response
  carries `playAvailable` for this. A game that is not ready to play shows no
  button. These links carry `game_share.public_page`: a leader's hand-made
  workshop share never reuses them, and the game's list of workshop shares leaves
  them out. The key owner's overview of game shares does list them. Switching the page off, unpublishing a game, changing the workshop
  key, and deleting the workshop revoke them; the next visit creates fresh ones
  with a fresh count.
- **Youth protection.** The page's links carry the workshop and its institution,
  so guests get the workshop's constraint, then the institution's, as for any
  workshop share (JUGENDSCHUTZ.md, 'Gäste'). The recorded author, used when
  neither sets one, is the workshop's creator, or else whoever assigned the
  workshop its key.
- **Privacy notes.** A short note that a game's name and description may become
  public, and should hold no real names, appears at both fields when a game is
  created, at the `public` switch, and in 'Öffentliche Seite'.

## Parts B and C: departures from the concept

- The page shows no game icons. The app displays icons nowhere else either.
- 'Kopieren' uses the create dialogue that 'All games' uses, not
  `POST /api/games/{id}/clone`, so the copy looks the same wherever it starts.

## Parts B and C: before the first real workshop

1. Walk through the screens in a browser: the 'Öffentliche Seite' section in
   both settings views, the page itself logged in and logged out, 'Spielen'
   until the count runs out, 'Herunterladen', and 'Kopieren' through a real
   Auth0 login and registration. The branch passed type checks, lint on the
   changed files, the build, and the Go unit and integration suites, but nobody
   has looked at the screens yet.
2. Start the backend once against a copy of the production database and check
   that every workshop received a slug.
3. Done, see Part A item 5.

## Small items

- A visitor who opens a revoked page link counts as a guess for the token lock,
  like any other share token that no longer exists.

- `en.json` spells 'Organizator: {{name}}'; the German text is correct.
- The keys `myOrganization.workshops.linkCopied` and `linkCopiedMessage` are no
  longer used.
- The lock's integration test runs only when asked, because a lock blocks every
  suite that follows in the same process:
  `TOKEN_LOCK_TEST=1 TOKEN_LOCK_MAX_FAILURES=20 TOKEN_LOCK_DURATION=3s go test -run TestTokenLockSuite`
  in `testing/`.
