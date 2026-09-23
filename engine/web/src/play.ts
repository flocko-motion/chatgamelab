/**
 * Watches a full session with no browser: same core the page uses, driven from
 * node. This is the integration harness — if the core needs a DOM to follow a
 * session, it is not headless.
 *
 *   node dist/play.js http://localhost:8080 standalone
 *
 * What it cannot do is speak. A live genre's audio runs over WebRTC between a
 * browser and the model, and node has neither a microphone nor a peer
 * connection. So this asserts everything up to the moment a player would join —
 * the wiring, the portrait, the gate opening, the invitation to connect — and
 * stops there rather than pretending to hold a conversation.
 */
import { Player } from "./player.js";
import { transcript } from "./state.js";
import { WebSocketTransport } from "./transport.js";

const [, , baseArg, sessionArg] = process.argv;
const base = baseArg ?? "http://localhost:8080";
const session = sessionArg ?? "standalone";

const sessionBase = new URL(`/sessions/${session}/`, base);
const url = new URL("live", sessionBase);
url.protocol = url.protocol.replace("http", "ws");

// No voice: this run watches rather than joins.
const player = new Player(new WebSocketTransport(url));

try {
  await player.restore(sessionBase);
  player.connect();
  await player.whenReady();

  // The init stem runs without anybody acting, which is the point of it: the
  // portrait is made and the gate opens before a player is asked for anything.
  await player.until((state) => state.image !== null, 10_000);
  await player.until((state) => state.started, 10_000);
  await player.until((state) => state.invited, 10_000);

  const state = player.state;
  console.log("--- session ---");
  console.log(`image:      ${state.image ? `${state.image.slice(0, 40)}…` : "(none)"}`);
  console.log(`started:    ${state.started}`);
  console.log(`invited:    ${state.invited}`);
  console.log(`utterances: ${state.utterances.length}`);
  console.log("--- transcript ---");
  console.log(transcript(state) || "(nobody has spoken; this run has no voice)");
  console.log("--- pipeline ---");
  for (const node of state.topology?.nodes ?? []) {
    console.log(`  ${node.role.padEnd(6)} ${node.name.padEnd(20)} ${state.phases[node.name] ?? "ready"}`);
  }

  // A genre that takes speech must say where it goes, or the graph is hiding
  // the one path the genre is about.
  const nodes = state.topology?.nodes ?? [];
  if (!nodes.some((node) => node.name === "player-input-audio")) {
    throw new Error("the wiring has nowhere for a player's voice to go");
  }
} finally {
  player.close();
}
