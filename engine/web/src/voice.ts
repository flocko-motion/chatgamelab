/**
 * The voice connection is the player's own: audio runs between this browser and
 * the model, and never through the engine.
 *
 * It is an interface for the same reason the transport is one — the core has to
 * stay playable with no DOM and no audio device. A browser supplies a real
 * implementation; a headless run supplies none and watches the session without
 * speaking, which is all node can honestly do with a voice genre.
 */
export interface Voice {
  /**
   * Opens the audio connection. Called when the engine says a player may
   * connect, which is after the game has begun — never on page load.
   */
  connect(): Promise<void>;
  close(): void;
  /**
   * Stops the player's voice reaching the character, or lets it through again.
   *
   * This is not a control anybody is offered: it is what a pause does while it
   * is still being asked for. Disabling a local track takes effect in this
   * browser at once, and somebody who has pressed pause should stop being heard
   * then rather than a round trip later.
   */
  setSending(on: boolean): void;
}

export type VoiceState = "idle" | "connecting" | "live" | "failed";
