/**
 * Watches a full session with no browser: same core the page uses, driven from
 * node. This is the integration harness — if the core needs a DOM to follow a
 * session, it is not headless.
 *
 *   node dist/play.js http://localhost:8080 standalone
 *   node dist/play.js <base> <session> --bad=own-key:high:images --choose=own-key:low:images,audio-live
 *
 * A session waiting for the player's choice is answered from the flags: --bad
 * is sent first and must be refused, the page is then rebuilt as a reload would
 * and must still be choosing, and --choose is the choice that starts the game.
 * A choice is an offer's ID, optionally with a tier and the toggles to turn on.
 *
 * What it cannot do is speak. A live genre's audio runs over WebRTC between a
 * browser and the model, and node has neither a microphone nor a peer
 * connection. So this asserts everything up to the moment a player would join —
 * the wiring, the portrait, the gate opening, the invitation to connect — and
 * stops there rather than pretending to hold a conversation.
 */
import { Player } from "./player.js";
import type { Choice } from "./protocol.js";
import { transcript } from "./state.js";
import { WebSocketTransport } from "./transport.js";

const positional = process.argv.slice(2).filter((arg) => !arg.startsWith("--"));
const flag = (name: string): string | undefined =>
  process.argv.find((arg) => arg.startsWith(`--${name}=`))?.slice(name.length + 3);
const [baseArg, sessionArg] = positional;
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

  console.log(`choosing:   ${player.state.choosing}`);
  if (player.state.choosing) {
    console.log(`offers:     ${player.state.options?.offers.length ?? 0}`);
    const bad = flag("bad");
    if (bad) {
      const [offer, choice] = parseChoice(bad);
      const accepted = await player.choose(sessionBase, offer, choice);
      console.log(`refused:    ${!accepted}`);
      if (accepted) throw new Error(`the engine accepted a choice outside the caveats: ${bad}`);
    }
    // A reload while choosing must find the dialog again.
    await player.rebuild(sessionBase);
    console.log(`reloaded:   choosing=${player.state.choosing}`);

    const wanted = flag("choose");
    if (!wanted) throw new Error("the session waits for a choice and none was given (--choose)");
    const [offer, choice] = parseChoice(wanted);
    if (!(await player.choose(sessionBase, offer, choice))) {
      throw new Error(`the choice was refused: ${player.state.problem}`);
    }
    await player.until((state) => state.settled !== null, 10_000);
    console.log(`settled:    ${player.state.settled?.choice.tier} ${JSON.stringify(player.state.settled?.choice.toggles)}`);
  }

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

/** Reads "own-key:low:images,audio-live" as that offer at low with those toggles on. */
function parseChoice(text: string): [string, Choice | undefined] {
  const [offer, tier, toggles] = text.split(":");
  if (!tier) return [offer ?? "", undefined];
  const on = new Set((toggles ?? "").split(",").filter(Boolean));
  return [
    offer ?? "",
    {
      tier,
      toggles: {
        images: on.has("images"),
        "audio-out": on.has("audio-out"),
        "audio-in": on.has("audio-in"),
        "audio-live": on.has("audio-live"),
      },
    },
  ];
}
