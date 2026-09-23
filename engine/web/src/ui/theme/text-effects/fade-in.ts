/**
 * FadeInText: words fade in one after another with a staggered delay.
 *
 * Each word's delay counts from when its span is created, so a word streamed
 * in late still waits out its place in the stagger, up to the cap.
 */
import type { TextRenderer } from "./index.js";

/** Seconds of stagger per word, and its cap. */
const STAGGER = 0.06;
const MAX_DELAY = 3;

export class FadeInText implements TextRenderer {
  #text = "";
  #element: HTMLElement;
  /** Words at even indices, the whitespace between them at odd ones. */
  #parts: string[] = [];
  #nodes: ChildNode[] = [];

  constructor(element: HTMLElement) {
    this.#element = element;
    element.replaceChildren();
  }

  update(text: string): void {
    if (text === this.#text) return;
    const grew = text.startsWith(this.#text);
    this.#text = text;
    const parts = text.split(/(\s+)/);

    // Growth can only extend the last word or the whitespace before it.
    const kept = Math.min(parts.length, this.#parts.length);
    for (let i = grew ? Math.max(0, kept - 2) : 0; i < kept; i++) {
      const part = parts[i] ?? "";
      if (part !== this.#parts[i]) this.#nodes[i]!.textContent = part;
    }
    for (const node of this.#nodes.splice(parts.length)) node.remove();

    for (let i = kept; i < parts.length; i++) {
      const part = parts[i] ?? "";
      let node: ChildNode;
      if (i % 2 === 1) {
        node = document.createTextNode(part);
      } else {
        const span = document.createElement("span");
        span.className = "fx-fade-in";
        span.style.animationDelay = `${Math.min((i / 2) * STAGGER, MAX_DELAY)}s`;
        span.textContent = part;
        node = span;
      }
      this.#nodes.push(node);
      this.#element.append(node);
    }
    this.#parts = parts;
  }

  dispose(): void {}
}
