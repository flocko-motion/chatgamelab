/**
 * The complete state of a session, owned by the core.
 *
 * This is what makes the player headless: everything the UI could draw is here,
 * so a test can play a full game and assert on this object without a DOM.
 */
import type { Phase, UsageReport } from "./protocol.js";
import type { VoiceState } from "./voice.js";

/**
 * "reconnecting" is distinct from "failed" because the socket puts itself back
 * together: one says the game is coming back, the other that it is not.
 */
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
  /**
   * What a player may supply — the genre's interaction model, which it states
   * outright. A genre offering no text box shows none.
   */
  readonly inputs: string[];
  /**
   * How many pictures this genre makes: "standing" for one that belongs to the
   * whole session, "per-turn" for one that belongs to a moment. It decides
   * where they go, which is a different question from how many arrive.
   */
  readonly imagery: string;
}

/** Where a session stands, for a client that was not watching it happen. */
export interface Snapshot {
  readonly started: boolean;
  readonly phases: Record<string, Phase>;
  readonly props: Record<string, string>;
  readonly usage: UsageReport;
  /**
   * Where a live genre's conversation stands. Absent for a genre that holds
   * none.
   *
   * The one thing a reload cannot be shown: the invitation to connect was sent
   * while this page did not exist, and a live connection cannot be replayed.
   */
  readonly conversation?: Conversation;
}

/**
 * "awaiting" is ready for a first connection, "open" a conversation this page
 * is not part of — a reload leaves one running with nobody on the other end —
 * and "paused" one let go, waiting to be picked up.
 */
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
  /**
   * What has gone wrong, in words a player can act on. Null when nothing has.
   *
   * Held apart from the connection states because those say which mechanism
   * failed, and this says what that means for whoever is looking at the page.
   */
  problem: string | null;
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
  /**
   * Whether the engine has invited this player to open a voice connection. It
   * follows the gate, so a portrait is on screen before anyone is asked to
   * speak — and until it arrives there is no conversation, and nothing being
   * billed for one.
   */
  invited: boolean;
  /** How the player's own audio connection is getting on. */
  voice: VoiceState;
  /**
   * Whether the conversation was let go for want of anybody speaking. A live
   * genre costs money for every second it stays open, so an idle one is closed
   * rather than left running — and picked up again on request, with the
   * character remembering what was said.
   */
  paused: boolean;
  /**
   * Whether this player has asked for the conversation to be let go, and the
   * engine has not confirmed it yet. Held apart from `paused`, which is the
   * engine's answer: closing a live session takes a round trip, and a control
   * that says nothing until it returns reads as a control that did nothing.
   */
  pausing: boolean;
  /** The same, the other way: a resume asked for and not yet live. */
  resuming: boolean;
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
  };
}

/**
 * Whether this genre has anywhere to type. Read off the wiring rather than
 * guessed: a live conversation has no node that takes typed words, because the
 * model has no event that would deliver them, so offering a text box would be
 * offering a control that cannot work.
 */
export function acceptsTypedInput(state: PlayerState): boolean {
  return accepts(state, "text");
}

/**
 * How this genre takes speech, if it does.
 *
 * The two want different controls, and offering the wrong one misleads rather
 * than merely looking odd: a pause control on a turn-based genre offers to stop
 * a conversation that is not running, and a hold-to-talk button on a
 * full-duplex one implies a turn that does not exist.
 */
export type VoiceMode = "full-duplex" | "push-to-talk" | null;

export function voiceMode(state: PlayerState): VoiceMode {
  if (accepts(state, "audio-full-duplex")) return "full-duplex";
  if (accepts(state, "audio-push-to-talk")) return "push-to-talk";
  return null;
}

/**
 * Where the conversation's audio stands, as the one value its control is drawn
 * from.
 *
 * The two transitions are states in their own right because both take a round
 * trip — one to let a session go, one to build a new peer connection — and they
 * last until the far end says the change happened rather than for a fixed
 * animation, which would lie whenever the engine took longer.
 *
 * Null while there is no control to offer: a genre that is not full-duplex has
 * none, and a conversation still waking up has nothing to press. Both are
 * things to say rather than buttons to show.
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
 * Whether the genre takes this kind of input, as it declares.
 *
 * Asked rather than worked out, which two earlier attempts here argue for.
 * Matching node names made a rename in a genre silently remove a control.
 * Reading the graph's shape — a source with an outgoing edge of that kind —
 * was worse, because it counted the values a genre seeds itself: the prompt
 * feeding a portrait read as somewhere the player could type.
 */
function accepts(state: PlayerState, kind: string): boolean {
  return state.topology?.inputs?.includes(kind) ?? false;
}

/** The transcript as plain text, which is what a headless assertion wants. */
export function transcript(state: PlayerState): string {
  return state.utterances.map((u) => u.text).join("\n");
}
