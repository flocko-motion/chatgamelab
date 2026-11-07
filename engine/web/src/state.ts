/** Everything the UI could draw, so a test can play a full game and assert on it without a DOM. */
import type { Options, Phase, Settled, Toggles, UsageReport } from "./protocol.js";
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

/** A node in the session's wiring, as fetched once at startup. */
export interface TopologyNode {
  readonly name: string;
  readonly role: "source" | "block" | "sink" | "gate";
}

export interface TopologyEdge {
  readonly from: string;
  readonly to: string;
  readonly kind: string;
}

export interface Topology {
  /** The wiring's name, which is the genre. */
  readonly name: string;
  /** What this game is called, as its author named it. Empty if nobody did. */
  readonly title?: string;
  readonly nodes: TopologyNode[];
  readonly edges: TopologyEdge[];
  /** What a player may supply, as the genre declares it. */
  readonly inputs: string[];
  /** "standing": one picture for the whole session; "per-turn": one per moment. */
  readonly imagery: string;
}

/** Where a session stands, for a client that was not watching it happen. */
export interface Snapshot {
  readonly started: boolean;
  readonly phases: Record<string, Phase>;
  readonly props: Record<string, string>;
  readonly usage: UsageReport;
  /** Where a live genre's conversation stands, which a reload cannot learn by replay. */
  readonly conversation?: Conversation;
  /** Whether the session waits for the player's choice. */
  readonly choosing: boolean;
  /** What the session runs on, once settled. */
  readonly settled?: Settled;
}

/** "open" is a conversation running with nobody on this end, as a reload leaves it. */
export type Conversation = "idle" | "awaiting" | "open" | "paused";

export interface PlayerState {
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
  /** What has gone wrong, in words a player can act on. */
  problem: string | null;
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

export function emptyState(): PlayerState {
  return {
    connection: "idle",
    utterances: [],
    props: {},
    image: null,
    audioChunks: 0,
    notes: [],
    problem: null,
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
export function allows(state: PlayerState, toggle: ToggleName): boolean {
  const settled = state.settled;
  if (!settled) return false;
  const offer = state.options?.offers.find((o) => o.offer === settled.offer);
  const tier = offer?.tiers.find((t) => t.tier === settled.choice.tier);
  return tier?.toggles[toggle] ?? false;
}

/** Whether this genre makes pictures at all, which is when a pictures switch means something. */
export function makesPictures(state: PlayerState): boolean {
  return (state.topology?.imagery ?? "") !== "";
}

/** Whether this genre has anywhere to type, as its wiring declares. */
export function acceptsTypedInput(state: PlayerState): boolean {
  return accepts(state, "text");
}

/** How this genre takes speech, which picks a pause control or hold-to-talk. */
export type VoiceMode = "full-duplex" | "push-to-talk" | null;

export function voiceMode(state: PlayerState): VoiceMode {
  if (accepts(state, "audio-full-duplex")) return "full-duplex";
  if (accepts(state, "audio-push-to-talk")) return "push-to-talk";
  return null;
}

/**
 * What the audio control draws. Each transition lasts until the far end confirms
 * the change, as both take a round trip; null while there is nothing to press.
 */
export type AudioState = "live" | "pausing" | "paused" | "resuming";

export function audioState(state: PlayerState): AudioState | null {
  if (voiceMode(state) !== "full-duplex") return null;
  if (state.pausing) return "pausing";
  if (state.resuming) return "resuming";
  if (state.paused) return "paused";
  if (state.voice === "live") return "live";
  return null;
}

/** Whether one picture stands for the session rather than for a moment in it. */
export function hasStandingImage(state: PlayerState): boolean {
  return state.topology?.imagery === "standing";
}

/**
 * Read from the declared inputs: node names break on a rename, and graph shape
 * mistakes a genre's self-seeded prompts for player input.
 */
function accepts(state: PlayerState, kind: string): boolean {
  return state.topology?.inputs?.includes(kind) ?? false;
}

/** The transcript as plain text, which is what a headless assertion wants. */
export function transcript(state: PlayerState): string {
  return state.utterances.map((u) => u.text).join("\n");
}
