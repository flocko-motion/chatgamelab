/**
 * The debug relay's end in the page, loaded on a dev server only: it records
 * the session from before the live connection opens, keeps trying to reach
 * the relay on this machine, and runs what the relay sends. A small mark in
 * the pipeline's corner says how the relay is: not reachable (grey, which is
 * no error: it runs only while somebody debugs), held by another tab (red),
 * or connected (green).
 */
import { DebugRecorder, evaluate } from "../debug.js";
import type { PlaybackEngine } from "../playback-engine.js";
import { text, type TextKey } from "../texts.js";
import type { TappedTransport } from "../transport.js";

/** Where the relay listens: `go/engine/cmd/debug-relay`. */
const RELAY = "ws://localhost:7199/connect";

/** The close code of a relay that holds another tab. */
const BUSY = 4409;

/** The longest wait between two tries, short as the relay is started once a session is stuck. */
const CEILING = 3_000;

/** What the relay sends: that the tab is now its own, and expressions to run. */
type FromRelay = { type: "welcome" } | { type: "eval"; id: number; expr: string };

export type RelayState = "connecting" | "busy" | "connected";

const LABELS: Record<RelayState, TextKey> = {
  connecting: "relay.connecting",
  busy: "relay.busy",
  connected: "relay.connected",
};

export interface Debugging {
  /** Says the mark's text again, in the language now in use. */
  relabel(): void;
}

/** What the tab tells the relay about itself. */
export interface TabLabel {
  game: string;
  session: string;
  started: string;
}

/**
 * Starts recording a session and connecting to the relay. Called once the
 * page knows it runs on a dev server, before the live connection opens.
 */
export function startDebugging(
  playback: PlaybackEngine,
  transport: TappedTransport,
  label: TabLabel,
  corner: HTMLElement,
): Debugging {
  const recorder = new DebugRecorder();
  recorder.record(transport, playback);
  addEventListener("error", (event) => recorder.error(event.error?.stack ?? event.message));
  addEventListener("unhandledrejection", (event) => recorder.error(String(event.reason?.stack ?? event.reason)));

  const scope = { playback, recorder, snapshot: () => recorder.snapshot(playback) };

  const mark = document.createElement("span");
  mark.className = "relay-mark";
  mark.setAttribute("role", "status");
  corner.append(mark);

  let state: RelayState = "connecting";
  const show = (next: RelayState): void => {
    state = next;
    mark.dataset["state"] = next;
    mark.textContent = text(LABELS[next]);
  };
  show("connecting");

  let attempt = 0;
  const connect = (): void => {
    const socket = new WebSocket(RELAY);
    socket.onopen = () => socket.send(JSON.stringify({ type: "hello", label }));
    socket.onmessage = (message: MessageEvent<string>) => {
      let msg: FromRelay;
      try {
        msg = JSON.parse(message.data) as FromRelay;
      } catch {
        return;
      }
      switch (msg.type) {
        case "welcome":
          attempt = 0;
          show("connected");
          break;
        case "eval":
          void evaluate(msg.expr, scope).then((result) => {
            socket.send(JSON.stringify({ type: "result", id: msg.id, ...result }));
          });
          break;
      }
    };
    // A relay that is not running fails the socket, which also closes it.
    socket.onclose = (event) => {
      show(event.code === BUSY ? "busy" : "connecting");
      setTimeout(connect, Math.min(500 * 2 ** attempt++, CEILING));
    };
  };
  connect();

  return { relabel: () => show(state) };
}
