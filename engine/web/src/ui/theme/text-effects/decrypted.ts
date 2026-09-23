/**
 * DecryptedText: scrambled characters that progressively reveal the real text.
 * Adapted from ReactBits (https://reactbits.dev/text-animations/decrypted-text)
 *
 * While streaming, only characters not yet revealed are scrambled, so each
 * delta arrives as noise and resolves behind the reveal front.
 */
import type { TextRenderer } from "./index.js";
import { pick } from "./text.js";

const CHARS =
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?";

/** ms between reveal steps. */
const SPEED = 20;
/** Characters revealed per step. */
const STEP = 2;

export class DecryptedText implements TextRenderer {
  #text = "";
  #revealed = 0;
  /** The revealed prefix, and the scrambled rest after it. */
  #clear = document.createTextNode("");
  #noise = document.createTextNode("");
  #timer: ReturnType<typeof setInterval> | null = null;

  constructor(element: HTMLElement) {
    element.replaceChildren(this.#clear, this.#noise);
  }

  update(text: string): void {
    if (text === this.#text) return;
    this.#text = text;
    if (text.length <= this.#revealed) {
      this.#revealed = text.length;
      this.#stop();
      this.#paint();
      return;
    }
    this.#paint();
    this.#timer ??= setInterval(() => this.#tick(), SPEED);
  }

  dispose(): void {
    this.#stop();
    this.#revealed = this.#text.length;
    this.#paint();
  }

  #tick(): void {
    this.#revealed = Math.min(this.#revealed + STEP, this.#text.length);
    this.#paint();
    if (this.#revealed >= this.#text.length) this.#stop();
  }

  #paint(): void {
    const prefix = this.#text.slice(0, this.#revealed);
    const shown = this.#clear.data;
    if (prefix.startsWith(shown)) this.#clear.appendData(prefix.slice(shown.length));
    else this.#clear.data = prefix;
    this.#noise.data = scramble(this.#text.slice(this.#revealed));
  }

  #stop(): void {
    if (this.#timer !== null) clearInterval(this.#timer);
    this.#timer = null;
  }
}

/** Every character noise except spaces and line breaks, which keep the shape. */
function scramble(text: string): string {
  let out = "";
  for (const char of text.split("")) out += char === " " || char === "\n" ? char : pick(CHARS);
  return out;
}
