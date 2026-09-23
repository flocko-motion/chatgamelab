/**
 * Plays a full session with no browser: same core the page uses, driven from
 * node. This is the integration harness — if a game cannot be played from here,
 * the player is not headless.
 *
 *   node dist/play.js http://localhost:8080 standalone
 */
import { Player } from "./player.js";
import { transcript } from "./state.js";
import { WebSocketTransport } from "./transport.js";

const [, , baseArg, sessionArg] = process.argv;
const base = baseArg ?? "http://localhost:8080";
const session = sessionArg ?? "standalone";

const url = new URL(`/sessions/${session}/live`, base);
url.protocol = url.protocol.replace("http", "ws");

const player = new Player(new WebSocketTransport(url));

// One frame of silence stands in for the microphone: the point is to exercise
// the round trip, not to say anything.
const frame = new Int16Array(2400).buffer;

try {
  player.connect();
  await player.whenReady();

  for (let turn = 1; turn <= 3; turn++) {
    player.takeFloor();
    player.speak(frame);
    await player.until((state) => state.utterances.length >= turn, 5_000);
  }

  const state = player.state;
  console.log("--- transcript ---");
  console.log(transcript(state));
  console.log("--- observer ---");
  console.log(state.flags.length ? state.flags.join("\n") : "(nothing flagged)");
  console.log("--- session ---");
  console.log(`image:       ${state.image ?? "(none)"}`);
  console.log(`audio chunks: ${state.audioChunks}`);
  console.log(`utterances:  ${state.utterances.length}`);

  if (state.utterances.length === 0) {
    throw new Error("no reply from the engine");
  }
} finally {
  player.close();
}
