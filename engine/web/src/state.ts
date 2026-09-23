/**
 * The complete state of a session, owned by the core.
 *
 * This is what makes the player headless: everything the UI could draw is here,
 * so a test can play a full game and assert on this object without a DOM.
 */
export type ConnectionState = "idle" | "connecting" | "open" | "closed" | "failed";

export interface Utterance {
  readonly id: number;
  /** Who is speaking. The player's own speech is never transcribed. */
  readonly speaker: "npc";
  text: string;
  /** Set once the utterance is known to be finished. */
  complete: boolean;
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
  };
}

/** The transcript as plain text, which is what a headless assertion wants. */
export function transcript(state: PlayerState): string {
  return state.utterances.map((u) => u.text).join("\n");
}
