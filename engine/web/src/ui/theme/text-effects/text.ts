/** Shared plumbing for the text effects: plain text and a cheap growing write. */
import type { TextRenderer } from "./index.js";

/** Writes `text` into `node`, appending when it only grew, as a streamed message does. */
export function write(node: Text, text: string): void {
  const shown = node.data;
  if (text === shown) return;
  if (text.startsWith(shown)) node.appendData(text.slice(shown.length));
  else node.data = text;
}

/** Read once per renderer: a message keeps the motion it started with. */
export function reducedMotion(): boolean {
  return typeof matchMedia === "function" && matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** A random character from `chars`. */
export function pick(chars: string): string {
  return chars.charAt(Math.floor(Math.random() * chars.length));
}

/** The text as it is, which is what `"none"` and reduced motion show. */
export class PlainText implements TextRenderer {
  #node = document.createTextNode("");

  constructor(element: HTMLElement) {
    element.replaceChildren(this.#node);
  }

  update(text: string): void {
    write(this.#node, text);
  }

  dispose(): void {}
}
