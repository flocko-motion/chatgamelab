/** Everything the UI could draw, so a test can play a full game and assert on it without a DOM. */
import type { Options, Settled, Toggles, Topology, UsageReport } from "./client.js";
import type { Phase, Problem, Refusal } from "./protocol.js";
import type { Notice, TextKey } from "./texts.js";
import type { VoiceState } from "./voice.js";

/** "reconnecting" means the game is coming back; "failed" means it is not. */
export type ConnectionState =
  | "idle"
  | "connecting"
  | "open"
  | "reconnecting"
  | "closed"
  | "failed";

export interface Utterance {
  readonly id: number;
  /** Who is speaking. The player's own speech is never transcribed. */
  readonly speaker: "npc";
  text: string;
  /** Set once the utterance is known to be finished. */
  complete: boolean;
}

/** "open" is a conversation running with nobody on this end, as a reload leaves it. */
export type Conversation = "idle" | "awaiting" | "open" | "paused";

export interface PlaybackEngineState {
  connection: ConnectionState;
  /** The conversation so far, oldest first. */
  utterances: Utterance[];
  /** Status values, for genres that track them. */
  props: Record<string, string>;
  /** The session's portrait, once it exists. */
  image: string | null;
  /** Audio chunks received; a headless run asserts on the count, not the sound. */
  audioChunks: number;
  /** Anything the client did not recognise, kept rather than dropped. */
  notes: string[];
  /** What has gone wrong, said so a player can act on it. */
  problem: Notice | null;
  /** Why the session could not be reached, when the engine refused it. */
  refusal: Refusal | null;
  /** A fault during play that leaves the controls working, such as a rejected key. */
  fault: Notice | null;
  /** The wiring, fetched once. Null until it arrives. */
  topology: Topology | null;
  /** What each block is doing right now, keyed by node name. */
  phases: Record<string, Phase>;
  /** What the session has spent, per block and per model. Null until reported. */
  usage: UsageReport | null;
  /** Whether the gate has released the player's inputs after preparation. */
  started: boolean;
  /** Whether the engine has invited a voice connection, which follows the gate. */
  invited: boolean;
  /** How the player's own audio connection is getting on. */
  voice: VoiceState;
  /** Whether the conversation was let go, as an idle live genre is to stop the billing. */
  paused: boolean;
  /** A pause asked for and awaiting the engine's answer in `paused`. */
  pausing: boolean;
  /** The same, the other way: a resume asked for and not yet live. */
  resuming: boolean;
  /** Whether the session waits for the player to choose how to play. */
  choosing: boolean;
  /** What the player may choose from. Null until fetched. */
  options: Options | null;
  /** What the session runs on, once it has settled. */
  settled: Settled | null;
}

export function emptyState(): PlaybackEngineState {
  return {
    connection: "idle",
    utterances: [],
    props: {},
    image: null,
    audioChunks: 0,
    notes: [],
    problem: null,
    refusal: null,
    fault: null,
    topology: null,
    phases: {},
    usage: null,
    started: false,
    invited: false,
    voice: "idle",
    paused: false,
    pausing: false,
    resuming: false,
    choosing: false,
    options: null,
    settled: null,
  };
}

/** The text that says what each problem means to a player. */
const PROBLEM_TEXT: Record<Problem, TextKey> = {
  no_key: "problems.noKey",
  genre: "problems.genre",
  no_plays: "problems.noPlays",
  key_rejected: "problems.keyRejected",
  unreachable: "problems.unreachable",
  stale: "problems.stale",
  unknown: "problems.unknown",
  denied: "problems.denied",
  failed: "problems.failed",
};

/** A refusal as a notice: its problem says what to tell the player, its message is the detail. */
export function explain(refusal: Refusal): Notice {
  const key = PROBLEM_TEXT[refusal.problem] ?? PROBLEM_TEXT.failed;
  return refusal.message ? { key, detail: refusal.message } : { key };
}

/** A toggle's name in its JSON form. */
export type ToggleName = keyof Toggles;

/** What each toggle means to a player. */
export const TOGGLE_LABELS: Record<ToggleName, string> = {
  images: "Pictures",
  "audio-out": "Narration",
  "audio-in": "Voice input",
  "audio-live": "Live voice",
};

/** Whether the settled offer allows a toggle at the settled tier, so it may go on. */
export function allows(state: PlaybackEngineState, toggle: ToggleName): boolean {
  const settled = state.settled;
  if (!settled) return false;
  const offer = state.options?.offers.find((o) => o.offer === settled.offer);
  const tier = offer?.tiers.find((t) => t.tier === settled.choice.tier);
  return tier?.toggles[toggle] ?? false;
}

/** Whether this genre makes pictures at all, which is when a pictures switch means something. */
export function makesPictures(state: PlaybackEngineState): boolean {
  return (state.topology?.imagery ?? "") !== "";
}

/** Whether this genre has anywhere to type, as its wiring declares. */
export function acceptsTypedInput(state: PlaybackEngineState): boolean {
  return accepts(state, "text");
}

/** How this genre takes speech, which picks a pause control or hold-to-talk. */
export type VoiceMode = "full-duplex" | "push-to-talk" | null;

export function voiceMode(state: PlaybackEngineState): VoiceMode {
  if (accepts(state, "audio-full-duplex")) return "full-duplex";
  if (accepts(state, "audio-push-to-talk")) return "push-to-talk";
  return null;
}

/**
 * What the audio control draws. Each transition lasts until the far end confirms
 * the change, as both take a round trip; null while there is nothing to press.
 */
export type AudioState = "live" | "pausing" | "paused" | "resuming";

export function audioState(state: PlaybackEngineState): AudioState | null {
  if (voiceMode(state) !== "full-duplex") return null;
  if (state.pausing) return "pausing";
  if (state.resuming) return "resuming";
  if (state.paused) return "paused";
  if (state.voice === "live") return "live";
  return null;
}

/** Whether one picture stands for the session rather than for a moment in it. */
export function hasStandingImage(state: PlaybackEngineState): boolean {
  return state.topology?.imagery === "standing";
}

/**
 * Read from the declared inputs: node names break on a rename, and graph shape
 * mistakes a genre's self-seeded prompts for player input.
 */
function accepts(state: PlaybackEngineState, kind: string): boolean {
  return state.topology?.inputs?.includes(kind) ?? false;
}

/** The transcript as plain text, which is what a headless assertion wants. */
export function transcript(state: PlaybackEngineState): string {
  return state.utterances.map((u) => u.text).join("\n");
}
