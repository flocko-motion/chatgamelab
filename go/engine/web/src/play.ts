/**
 * Watches a full session with no browser: same core the page uses, driven from
 * node. This is the integration harness — if the core needs a DOM to follow a
 * session, it is not headless.
 *
 *   node dist/play.js <origin> <session>
 *   node dist/play.js <origin> --new=game:bridge --once=k1
 *   node dist/play.js <origin> <session> --bad=own-key:high:images --choose=own-key:low:images,audio-live
 *
 * The origin serves the engine at /api/engine/…: the platform, or the dev server.
 *
 * Opened with --new it creates the session the way the page does, asking a
 * second time with the same one-time key to see the same session answer, and
 * rebuilds as a reload would to see it resume.
 *
 * A session waiting for the player's choice is answered from the flags: --bad
 * is sent first and must be refused, the page is then rebuilt as a reload would
 * and must still be choosing, and --choose is the choice that starts the game.
 * A choice is an offer's ID, optionally with a tier and the toggles to turn on.
 *
 * What it cannot do is hold a live conversation. A live genre's audio runs
 * over WebRTC between a browser and the model, and node has neither a
 * microphone nor a peer connection. So this asserts everything up to the moment
 * a player would join — the wiring, the portrait, the gate opening, the
 * invitation to connect — and stops there rather than pretending to.
 *
 * A push-to-talk genre takes a clip over the session's own connection, so with
 * --speak it sends one, a tone the mock platform hears by its length, and
 * waits for the player's line and the reply:
 *
 *   node dist/play.js <origin> <session> --speak
 */
import { engineClient, liveURL, type Choice, type SessionRef } from "./client.js";
import { create, transientMemory, type Selector } from "./launch.js";
import { PlaybackEngine } from "./playback-engine.js";
import { holdsToTalk, transcript, voiceMode } from "./state.js";
import { NoticeError, describe } from "./texts.js";
import { WebSocketTransport } from "./transport.js";

const positional = process.argv.slice(2).filter((arg) => !arg.startsWith("--"));
const flag = (name: string): string | undefined =>
  process.argv.find((arg) => arg.startsWith(`--${name}=`))?.slice(name.length + 3);
const [originArg, sessionArg] = positional;
const origin = originArg ?? "http://localhost:8080";
const client = engineClient(origin);
const wanted = flag("new");
const id = wanted ? await createSession(wanted) : sessionArg;
if (!id) throw new Error("name a session, or --new=game:<id> to create one");
const sessionRef: SessionRef = { client, id };

// No voice: this run watches rather than joins.
const playback = new PlaybackEngine(new WebSocketTransport(liveURL(sessionRef, origin)));

try {
  await playback.restore(sessionRef);
  if (playback.state.refusal) throw new Error(`refused: ${playback.state.refusal.problem}`);
  playback.connect();
  await playback.whenReady();
  if (wanted) {
    // A reload finds the session the address now names.
    await playback.rebuild(sessionRef);
    console.log(`resumed:    ${playback.state.topology !== null}`);
  }

  console.log(`choosing:   ${playback.state.choosing}`);
  if (playback.state.choosing) {
    console.log(`offers:     ${playback.state.options?.offers.length ?? 0}`);
    const bad = flag("bad");
    if (bad) {
      const [offer, choice] = parseChoice(bad);
      const accepted = await playback.choose(sessionRef, offer, choice);
      console.log(`refused:    ${!accepted}`);
      if (accepted) throw new Error(`the engine accepted a choice outside the caveats: ${bad}`);
    }
    // A reload while choosing must find the dialog again.
    await playback.rebuild(sessionRef);
    console.log(`reloaded:   choosing=${playback.state.choosing}`);

    const wanted = flag("choose");
    if (!wanted) throw new Error("the session waits for a choice and none was given (--choose)");
    const [offer, choice] = parseChoice(wanted);
    if (!(await playback.choose(sessionRef, offer, choice))) {
      throw new Error(`the choice was refused: ${playback.state.problem ? describe(playback.state.problem) : "?"}`);
    }
    await playback.until((state) => state.settled !== null, 10_000);
    console.log(`settled:    ${playback.state.settled?.choice.tier} ${JSON.stringify(playback.state.settled?.choice.toggles)}`);
  }

  // The init stem runs without anybody acting, which is the point of it: the
  // portrait is made and the gate opens before a player is asked for anything.
  await playback.until((state) => state.image !== null, 10_000);
  await playback.until((state) => state.started, 10_000);
  if (voiceMode(playback.state) === "full-duplex") await playback.until((state) => state.invited, 10_000);

  if (process.argv.includes("--speak")) {
    if (!holdsToTalk(playback.state)) throw new Error("--speak: the session takes no push-to-talk");
    // The opening's prose first, as a player would wait for it.
    await playback.until((state) => state.utterances.some((u) => u.speaker === "npc" && u.text !== ""), 10_000);
    playback.speak(spokenClip(1.5));
    const heard = await playback.until((state) => state.utterances.some((u) => u.speaker === "player"), 10_000);
    console.log(`said:       ${heard.utterances.find((u) => u.speaker === "player")?.text}`);
    await playback.until((state) => state.utterances.at(-1)?.speaker === "npc" && state.utterances.at(-1)?.text !== "", 10_000);
    console.log(`replied:    true`);
  }

  const state = playback.state;
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
  playback.close();
}

/** Creates a session as the page does for ?new&game=…, and asks again with the same key. */
async function createSession(wanted: string): Promise<string> {
  const [kind, ...rest] = wanted.split(":");
  if (kind !== "game" && kind !== "share") throw new Error(`--new=${wanted}: name game:<id> or share:<token>`);
  const selector: Selector = { kind, id: rest.join(":") };
  const pending = transientMemory();
  const once = flag("once");

  const created = await create(client, selector, "en", pending, once);
  if ("notice" in created) {
    console.log(`refused:    ${describe(created.notice)}`);
    throw new NoticeError(created.notice);
  }
  console.log(`created:    ${created.id}`);
  if (once) {
    const again = await create(client, selector, "en", pending, once);
    console.log(`repeated:   ${"id" in again && again.id === created.id ? "same" : "different"}`);
  }
  return created.id;
}

/** A clip of a tone lasting seconds, PCM16 at 24 kHz mono, as the page records speech. */
function spokenClip(seconds: number): ArrayBuffer {
  const rate = 24_000;
  const samples = new Int16Array(Math.round(seconds * rate));
  for (let i = 0; i < samples.length; i++) {
    samples[i] = Math.round(Math.sin((2 * Math.PI * 220 * i) / rate) * 0.3 * 32767);
  }
  return samples.buffer;
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
