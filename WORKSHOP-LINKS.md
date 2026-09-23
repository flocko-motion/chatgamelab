# Workshop links: status and follow-up

Branch `feat/memorable-workshop-links` implements Part A of the concept
'Workshop-Links merkbar machen + öffentliche Workshop-Seite'. This file records
what the branch contains, where it departs from the concept, and what remains.

## Built

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
- **Dead invites.** Asking for the invite link of a workshop whose pending
  invite has expired or is used up now marks that invite expired and creates a
  fresh one (`db.CreateWorkshopInvite`).
- **Token lock.** Lookups of invite, participant and play tokens that were never
  issued count as guesses (`server/tokenlock`). More than 300 within ten minutes
  lock every token check, correct ones included, for five minutes: `429`,
  `Retry-After`, code `token_locked`. `TOKEN_LOCK_MAX_FAILURES`,
  `TOKEN_LOCK_WINDOW` and `TOKEN_LOCK_DURATION` override the defaults. Tokens of
  deleted users, deleted workshops or reset access never count, and the backend
  drops a session cookie whose token no longer exists.
- **Share popup.** One component (`web/src/common/components/share`) for invite
  and re-login links: copyable link, the words in large type, a QR code that
  opens full screen, always black on white. Heads and staff get a QR button next
  to 'Organisator' in the workshop header.

## Departures from the concept

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

## Before the first real workshop

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
5. Run `./run-translate.sh` for the 36 languages beyond German and English. It
   also refreshes `server/lang/locales`, which was already out of step with the
   web copy before this branch.

## Open decisions

- **Who may make a game public.** Only its creator can switch `public` on; heads
  and staff can only switch it off (`server/db/game_writes.go:163`). The concept
  assumed heads and staff could publish participants' games. Decided in review:
  leave this rule alone for now.
- **Scaling the backend.** The lock lives in memory, which suits the single
  backend container in `docker-compose.yml`. More than one instance would need
  the counter in Postgres or a shared cache.

## Part B: public workshop page (not started)

Everything in the concept's Part B is open. Facts established while reviewing
it:

- `workshop.public` exists with the intended meaning, but nothing lists public
  workshops and no screen sets the flag (`server/db/schema.sql:61`).
- The next migration number is 033. New columns go into both a migration and
  `server/db/schema.sql`, because a fresh database starts from the schema file.
- The YAML export (`server/api/routes/games_yaml.go`) contains no API key IDs,
  sponsor IDs or creators. It does contain both system prompts, which is the
  game itself; decide whether a public download should include them.
- 'Spiel kopieren' needs a return path after login. `/auth/login` accepts no
  `redirect` parameter today, and production logs in through Auth0.
- `POST /api/games/{id}/clone` exists and clones public games.
- `game_share.remaining` already counts down on every session
  (`server/db/game_shares.go`), so a finite quota for public play links needs no
  new table.
- `/w/` collides with no existing route. It must join `isPublicRoute` in
  `web/src/routes/__root.tsx`, which also exempts it from the participant
  redirect.
- The word generator is in place for slug suffixes, and the share popup for the
  page's link and QR code.

## Small items

- `en.json` spells 'Organizator: {{name}}'; the German text is correct.
- The keys `myOrganization.workshops.linkCopied` and `linkCopiedMessage` are no
  longer used.
- The lock's integration test runs only when asked, because a lock blocks every
  suite that follows in the same process:
  `TOKEN_LOCK_TEST=1 TOKEN_LOCK_MAX_FAILURES=20 TOKEN_LOCK_DURATION=3s go test -run TestTokenLockSuite`
  in `testing/`.
