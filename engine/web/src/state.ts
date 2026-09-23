/**
 * The complete state of a session, owned by the core.
 *
 * This is what makes the player headless: everything the UI could draw is here,
 * so a test can play a full game and assert on this object without a DOM.
 */
import type { Phase, UsageReport } from "./protocol.js";

export type ConnectionState = "idle" | "connecting" | "open" | "closed" | "failed";

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
  readonly name: string;
  readonly nodes: TopologyNode[];
  readonly edges: TopologyEdge[];
}

/** Where a session stands, for a client that was not watching it happen. */
export interface Snapshot {
  readonly started: boolean;
  readonly phases: Record<string, Phase>;
  readonly props: Record<string, string>;
  readonly usage: UsageReport;
}

export interface PlayerState {
  connection: ConnectionState;
  /** The conversation so far, oldest first. */
  utterances: Utterance[];
  /** What the observer flagged, in order. */
  flags: string[];
  /** Status values, for genres that track them. */
  props: Record<string, string>;
  /** The session's portrait, once it exists. */
  image: string | null;
  /** Audio chunks received; a headless run asserts on the count, not the sound. */
  audioChunks: number;
  /** Anything the client did not recognise, kept rather than dropped. */
  notes: string[];
  /** The wiring, fetched once. Null until it arrives. */
  topology: Topology | null;
  /** What each block is doing right now, keyed by node name. */
  phases: Record<string, Phase>;
  /** What the session has spent, per block and per model. Null until reported. */
  usage: UsageReport | null;
  /**
   * Whether the game has begun. False while the init stem is still running:
   * the gate holds the player's inputs until preparation finishes, so a control
   * that looks usable before then is lying about what will happen.
   */
  started: boolean;
}

export function emptyState(): PlayerState {
  return {
    connection: "idle",
    utterances: [],
    flags: [],
    props: {},
    image: null,
    audioChunks: 0,
    notes: [],
    topology: null,
    phases: {},
    usage: null,
    started: false,
  };
}

/** The transcript as plain text, which is what a headless assertion wants. */
export function transcript(state: PlayerState): string {
  return state.utterances.map((u) => u.text).join("\n");
}
