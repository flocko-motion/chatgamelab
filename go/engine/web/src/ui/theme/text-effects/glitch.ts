/**
 * GlitchText: a few characters at a time corrupt, then restore, on every
 * other tick — a continuous, subtle glitch.
 * Inspired by ReactBits (https://reactbits.dev/text-animations/glitch-text)
 */
import type { TextRenderer } from "./index.js";
import { pick, write } from "./text.js";

const GLITCH_CHARS = "!@#$%^&*()_+-=[]{}|;:<>?/\\~`░▒▓█▄▀■□▪▫";

/** ms between ticks. */
const INTERVAL = 200;
/** Fraction of the characters corrupted per glitch. */
const INTENSITY = 0.02;

export class GlitchText implements TextRenderer {
  #text = "";
  #node = document.createTextNode("");
  /** Offsets currently showing a glitch character. */
  #corrupted: number[] = [];
  #glitched = false;
  #timer: ReturnType<typeof setInterval>;

  constructor(element: HTMLElement) {
    element.replaceChildren(this.#node);
    this.#timer = setInterval(() => this.#tick(), INTERVAL);
  }

  update(text: string): void {
    if (text === this.#text) return;
    this.#restore();
    write(this.#node, text);
    this.#text = text;
  }

  dispose(): void {
    clearInterval(this.#timer);
    this.#restore();
  }

  #tick(): void {
    if (this.#glitched) this.#restore();
    else this.#glitch();
    this.#glitched = !this.#glitched;
  }

  #glitch(): void {
    const length = this.#text.length;
    if (length === 0) return;
    const count = Math.max(1, Math.floor(length * INTENSITY));
    for (let i = 0; i < count; i++) {
      const at = Math.floor(Math.random() * length);
      const char = this.#text.charAt(at);
      if (char === " " || char === "\n") continue;
      this.#node.replaceData(at, 1, pick(GLITCH_CHARS));
      this.#corrupted.push(at);
    }
  }

  #restore(): void {
    for (const at of this.#corrupted) this.#node.replaceData(at, 1, this.#text.charAt(at));
    this.#corrupted = [];
  }
}
