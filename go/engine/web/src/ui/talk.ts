/**
 * Push-to-talk: the player holds the button, or Space while not typing, and
 * releases to send what they said as one clip. Sliding off the button cancels.
 * A recording stops and sends at LONGEST_MS; a press shorter than SHORTEST_MS
 * sends nothing, as it holds no words.
 *
 * It owns the button's recording state and its ring; whether the button shows
 * at all is the view's.
 */
import type { AudioIO } from "./audio.js";
import { text } from "../texts.js";

const SAMPLE_RATE = 24_000;
const SHORTEST_MS = 300;
const LONGEST_MS = 30_000;

export interface TalkHandlers {
  /** One clip of PCM16 at 24 kHz mono, the whole utterance. */
  onClip(clip: ArrayBuffer): void;
  /** The microphone was refused, or there is none. */
  onDenied(): void;
}

export class HoldToTalk {
  #frames: ArrayBuffer[] = [];
  /** Set while a press lasts, from its start. */
  #started: number | null = null;
  /** The capture starting, which a release may outrun on the first press. */
  #capturing: Promise<boolean> | null = null;
  #tick = 0;
  /** Stops the recording at the longest, even in a tab the browser stopped painting. */
  #cutoff: ReturnType<typeof setTimeout> | null = null;

  constructor(
    private readonly button: HTMLButtonElement,
    private readonly audio: AudioIO,
    private readonly handlers: TalkHandlers,
    typing: () => boolean,
  ) {
    this.#paint();
    button.addEventListener("pointerdown", (event) => {
      if (event.button !== 0 || button.disabled) return;
      // Kept from becoming a click or a text selection, which a long press is on touch.
      event.preventDefault();
      // A touch holds on to the button it began on; released, sliding off leaves it and cancels.
      if (button.hasPointerCapture(event.pointerId)) button.releasePointerCapture(event.pointerId);
      void this.#start();
    });
    button.addEventListener("pointerup", () => void this.#finish(true));
    button.addEventListener("pointerleave", () => void this.#finish(false));
    button.addEventListener("pointercancel", () => void this.#finish(false));
    button.addEventListener("contextmenu", (event) => event.preventDefault());

    document.addEventListener("keydown", (event) => {
      if (event.code !== "Space" || event.repeat || typing() || button.disabled || !this.#visible()) return;
      event.preventDefault();
      void this.#start();
    });
    document.addEventListener("keyup", (event) => {
      if (event.code !== "Space" || this.#started === null) return;
      event.preventDefault();
      void this.#finish(true);
    });
  }

  /** Drops a recording in progress, as when voice input goes off. */
  cancel(): void {
    void this.#finish(false);
  }

  /** Says the button's words again, as after the language changed. */
  refresh(): void {
    if (this.#started === null) this.#paint();
  }

  #visible(): boolean {
    return this.button.style.display !== "none";
  }

  async #start(): Promise<void> {
    if (this.#started !== null) return;
    this.#started = performance.now();
    this.#frames = [];
    this.#paint();
    this.#tick = requestAnimationFrame(() => this.#progress());
    this.#cutoff = setTimeout(() => void this.#finish(true), LONGEST_MS);

    this.#capturing = this.audio
      .startCapture((frame) => this.#frames.push(frame))
      .then(() => true)
      .catch(() => false);
    const started = await this.#capturing;
    if (!started) {
      // Asked on the first press; refused, the player can still type.
      this.#reset();
      this.handlers.onDenied();
      return;
    }
    // Released while the browser was still asking for the microphone.
    if (this.#started === null) this.audio.stopCapture();
  }

  /** Fills the ring towards the longest recording. */
  #progress(): void {
    if (this.#started === null) return;
    const elapsed = performance.now() - this.#started;
    this.button.style.setProperty("--progress", String(Math.min(elapsed / LONGEST_MS, 1)));
    this.#tick = requestAnimationFrame(() => this.#progress());
  }

  async #finish(send: boolean): Promise<void> {
    if (this.#started === null) return;
    const capturing = this.#capturing;
    const frames = this.#frames;
    this.#reset();
    if (!(await capturing)) return;
    this.audio.stopCapture();

    const bytes = frames.reduce((sum, frame) => sum + frame.byteLength, 0);
    const ms = (bytes / 2 / SAMPLE_RATE) * 1000;
    if (!send || ms < SHORTEST_MS) return;
    const clip = new Uint8Array(bytes);
    let at = 0;
    for (const frame of frames) {
      clip.set(new Uint8Array(frame), at);
      at += frame.byteLength;
    }
    this.handlers.onClip(clip.buffer);
  }

  #reset(): void {
    cancelAnimationFrame(this.#tick);
    if (this.#cutoff !== null) clearTimeout(this.#cutoff);
    this.#cutoff = null;
    this.#started = null;
    this.#capturing = null;
    this.#frames = [];
    this.#paint();
  }

  /** The button says whether it records, in its look and in words. */
  #paint(): void {
    const recording = this.#started !== null;
    this.button.dataset.state = recording ? "recording" : "idle";
    this.button.style.setProperty("--progress", "0");
    const label = text(recording ? "talk.recording" : "talk.hold");
    this.button.setAttribute("aria-label", label);
    this.button.title = label;
    this.button.setAttribute("aria-pressed", String(recording));
  }
}
