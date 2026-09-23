/**
 * The effects that are pure CSS: the text sits in a span whose class carries
 * the animation, and the renderer only keeps its text current. A one-shot
 * animation (ink bleed, parchment burn) plays once, from when the span is
 * created, so text streamed in afterwards arrives already settled.
 */
import type { TextRenderer } from "./index.js";
import { write } from "./text.js";

/** Text in one span of the given class. */
export class StyledText implements TextRenderer {
  #node = document.createTextNode("");

  constructor(element: HTMLElement, className: string) {
    const span = document.createElement("span");
    span.className = className;
    span.append(this.#node);
    element.replaceChildren(span);
  }

  update(text: string): void {
    write(this.#node, text);
  }

  dispose(): void {}
}

/** CRT monitor: phosphor glow and flicker on the text, scanlines laid over it. */
export class RetroText implements TextRenderer {
  #node = document.createTextNode("");

  constructor(element: HTMLElement) {
    const wrapper = document.createElement("span");
    wrapper.className = "fx-retro";
    const text = document.createElement("span");
    text.className = "fx-retro-text";
    text.append(this.#node);
    const scanlines = document.createElement("span");
    scanlines.className = "fx-retro-scanlines";
    scanlines.setAttribute("aria-hidden", "true");
    wrapper.append(text, scanlines);
    element.replaceChildren(wrapper);
  }

  update(text: string): void {
    write(this.#node, text);
  }

  dispose(): void {}
}
