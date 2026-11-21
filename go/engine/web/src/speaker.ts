/**
 * The speaker plays what the engine says aloud: a spoken reply's audio, frame
 * by frame as it arrives over the session's connection.
 *
 * An interface for the same reason the voice is one: the core stays playable
 * with no audio device. A browser supplies a real speaker; a headless run
 * supplies none and counts the frames.
 */
export interface Speaker {
  /** Queues one frame of 24 kHz mono PCM16 after the frames before it. */
  play(pcm: ArrayBuffer): void;
  /** Stops what is playing and drops what is queued. */
  discard(): void;
}
