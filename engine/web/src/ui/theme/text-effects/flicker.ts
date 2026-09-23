/**
 * FlickerText: random letters briefly dim or vanish, like a dying lightbulb,
 * on every other tick.
 */
import type { TextRenderer } from "./index.js";
import { write } from "./text.js";

/** ms between ticks. */
const INTERVAL = 400;
/** Fraction of the characters dimmed per flicker. */
const INTENSITY = 0.03;

export class FlickerText implements TextRenderer {
  #text = "";
  #element: HTMLElement;
  /** The whole text; detached while a flicker has split it into spans. */
  #node = document.createTextNode("");
  #flickered = false;
  #timer: ReturnType<typeof setInterval>;

  constructor(element: HTMLElement) {
    this.#element = element;
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
    if (this.#flickered) this.#restore();
    else this.#flicker();
    this.#flickered = !this.#flickered;
  }

  #flicker(): void {
    const text = this.#text;
    if (text.length === 0) return;
    const count = Math.max(1, Math.floor(text.length * INTENSITY));
    // Offset → off (opacity 0) or dim (0.3); a repeated offset keeps its last draw.
    const dimmed = new Map<number, boolean>();
    for (let i = 0; i < count; i++) {
      const at = Math.floor(Math.random() * text.length);
      const char = text.charAt(at);
      if (char !== " " && char !== "\n") dimmed.set(at, Math.random() < 0.5);
    }
    if (dimmed.size === 0) return;

    const parts: Node[] = [];
    let from = 0;
    for (const at of [...dimmed.keys()].sort((a, b) => a - b)) {
      if (at > from) parts.push(document.createTextNode(text.slice(from, at)));
      const span = document.createElement("span");
      span.className = dimmed.get(at) ? "fx-flicker-off" : "fx-flicker-dim";
      span.textContent = text.charAt(at);
      parts.push(span);
      from = at + 1;
    }
    if (from < text.length) parts.push(document.createTextNode(text.slice(from)));
    this.#element.replaceChildren(...parts);
  }

  #restore(): void {
    if (this.#node.parentNode !== this.#element) this.#element.replaceChildren(this.#node);
  }
}
