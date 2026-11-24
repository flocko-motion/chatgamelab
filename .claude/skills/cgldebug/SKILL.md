---
name: cgldebug
description: Look into a live, stuck playback-engine session in the developer's browser through the debug relay, without reloading it. Use when the user says a game or engine session is stuck, hangs, shows nothing, or behaves oddly in the browser, or runs /cgldebug.
---

# Debug a live engine session

A game opened on a dev server (the dev stack, or `make dev` in `go/engine`)
records its session from the start and keeps trying to reach the debug relay
on `localhost:7199`. Through the relay you read that tab's state and recent
history, and run JavaScript in it. **Never reload or reconnect the tab**: that
destroys exactly what you came to read.

## 1. Have a relay

```sh
curl -s localhost:7199/tab
```

- **No answer** (connection refused): start the relay in the background, from
  `go/engine`, with `run_in_background`:
  `go run ./cmd/debug-relay`. Then poll `/tab` every second or two for up to
  half a minute: the page retries every few seconds.
- **503 `no tab connected`**: no dev tab has reached the relay yet. Wait as
  above; if none comes, ask the user whether the game runs on a dev server
  (only there does the Insights pane's pipeline show a "Relay" mark in its
  corner).
- **200**: the label of the connected tab: `game`, `session`, `started`.

Tell the user which game and session is connected and ask them to confirm it
is the stuck one. The relay holds one tab; another tab's mark shows red
("in use by another tab"). If the wrong tab holds it, the user closes that tab
and the right one takes over within a few seconds.

## 2. Read the snapshot

```sh
curl -s localhost:7199/snapshot > /tmp/cgl-snapshot.json
```

It holds:

- `state`: the playback engine's whole state (`src/state.ts`,
  `PlaybackEngineState`): `connection`, `phases` per block, `runs`,
  `started`, `invited`, `voice`, `paused`, `pausing`, `resuming`,
  `choosing`, `problem`, `fault`, `refusal`, `utterances`, `topology`,
  `journal`. The picture is given by its size only.
- `state.journal`: **the engine's own record of the session, read it
  first.** It lives on the server, so it covers what happened before this
  page loaded, across every reload. Each entry has `time`, `source` (a block's
  name, `session`, or `client`), `level` (`info`/`warning`/`error`), `kind`
  and `detail`. It shows pages attaching and detaching (a reload is a
  `detached`/`attached` pair), offers, the live conversation's `standing`
  (idle → awaiting → open → paused), `opened`/`started`/`closed` with the
  provider's reason, how each conversation `ended`, every block's `phase`,
  failed calls, and each finished utterance.
- `buffer`: the last 500 entries, oldest first, each with an ISO `time` and a
  `kind`: `received` / `sent` (a text frame in full in `text`, an audio frame
  by its `bytes`), `phase` (`node`, `from`, `to`), `run` (`node`, `to`:
  started or done), `connection` (`from`, `to`), `notice` (`to`: which notice
  and its key, or cleared), `error` (`text`: an error the page ran into).

Read it with `jq`. Useful first questions:

- What do the journal's warnings and errors say, and when?
  `jq -r '.state.journal[] | select(.level != "info") | [.time,.source,.kind,.detail] | @tsv'`
- How does the journal end? `jq -r '.state.journal[-30:][] | [.time,.source,.level,.kind,.detail] | @tsv'`

- Which blocks are `working` and since when (their last `phase` entry)?
- What was the last frame received, and how long ago? Did the player's last
  `sent` get an answer?
- Is `connection` open? Any `reconnecting` in the buffer?
- Is a `problem`, `fault` or `error` set or logged?
- Does the state contradict itself (`pausing` stuck true, `hearing` with no
  clip pending, `started` false after the gate ran)?

## 3. Dig with eval

```sh
curl -s --data-binary 'playback.state.phases' localhost:7199/eval
curl -s --data-binary 'await fetch("/api/engine/sessions/" + new URLSearchParams(location.search).get("session") + "/state").then(r => r.json())' localhost:7199/eval
```

The body is an expression, or statements ending in `return`; `await` works.
In scope: `playback` (the `PlaybackEngine`), `recorder` (the
`DebugRecorder`), `snapshot()`, and the page's globals (`document`,
`location`). The result comes back as JSON; a throw comes back as 422 with
`{"error": …}`, a tab that takes over 10 s as 504.

Read, don't act: an expression that sends input, pauses or reconnects
changes the session under investigation. Ask the user before doing that.

## 4. Check the server's side

Compare with the engine's view of the same session and the same minutes:

- The engine's own snapshot: `GET /api/engine/sessions/<id>/state`,
  `/history` and `/journal` (through `eval` with `fetch`, as above, so it
  reaches the same origin). The journal is stored with the session and
  survives restarts; the store holds every entry (`game_session_event`, stream
  `journal`), the endpoint the latest 2000.
- The logs: on the dev stack, find the backend container with `docker ps`
  and read `docker logs --since <time> <container>`, from a little before the
  buffer's first suspicious entry. The standalone engine server (`make dev`)
  logs to the terminal it runs in; ask the user for that output.

## 5. Report

Say what is stuck, which side it is stuck on (page, connection, engine
block), the evidence from the snapshot and the logs with times, and the cause
if found. Propose the fix; leave the relay running unless the user says
otherwise, and say it is running.
