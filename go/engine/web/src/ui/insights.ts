/**
 * Insights: the workings of the game beside it, for every player. The header's
 * toggle opens and closes them, remembered per browser. On a wide screen they
 * are a side panel with the pipeline and the details; on a phone a bottom pane
 * with a feed of the blocks that ran, each opening its details full screen.
 * Which of the two shows is the stylesheet's business, by the screen's width;
 * this only keeps the state both read.
 */
import type { PlaybackEngineState, Run } from "../state.js";
import { blockLabel, text } from "../texts.js";

export interface InsightsElements {
  toggle: HTMLButtonElement;
  label: HTMLElement;
  panel: HTMLElement;
  runsTitle: HTMLElement;
  runs: HTMLOListElement;
  close: HTMLButtonElement;
}

export interface InsightsHandlers {
  /** A run in the feed was picked: its block's details are wanted. */
  onRun(node: string): void;
  /** The full-screen details were closed. */
  onClose(): void;
}

const REMEMBERED = "engine.insights";

/** Where the side panel stops and the phone's bottom pane begins (styles.css). */
const PHONE = "(max-width: 47.99rem)";

export class Insights {
  #shown = new Map<number, HTMLLIElement>();

  constructor(
    private readonly elements: InsightsElements,
    private readonly handlers: InsightsHandlers,
  ) {
    elements.toggle.addEventListener("click", () => this.#setOpen(!this.#open(), true));
    elements.close.addEventListener("click", () => {
      this.#setDetail(false);
      handlers.onClose();
    });
    this.#setOpen(remembered() ?? !matchMedia(PHONE).matches, false);
    this.relabel();
  }

  /** Says the texts again, in the language now in use. */
  relabel(): void {
    const { label, runsTitle, close } = this.elements;
    label.textContent = text("insights.label");
    this.#nameToggle();
    runsTitle.textContent = text("insights.feed");
    close.setAttribute("aria-label", text("insights.close"));
    close.title = text("insights.close");
    this.#shown.clear();
    this.elements.runs.replaceChildren();
  }

  /** Adds the runs not yet in the feed, and marks the finished ones. */
  render(state: PlaybackEngineState): void {
    const list = this.elements.runs;
    const kept = new Set(state.runs.map((r) => r.id));
    for (const [id, item] of this.#shown) {
      if (!kept.has(id)) {
        item.remove();
        this.#shown.delete(id);
      }
    }
    for (const run of state.runs) {
      const item = this.#shown.get(run.id) ?? this.#add(run);
      item.dataset["done"] = String(run.done);
    }
    if (state.runs.length === 0 && !list.querySelector(".runs-empty")) {
      const empty = document.createElement("li");
      empty.className = "runs-empty";
      empty.textContent = text("insights.empty");
      list.append(empty);
    } else if (state.runs.length > 0) {
      list.querySelector(".runs-empty")?.remove();
    }
  }

  #add(run: Run): HTMLLIElement {
    const item = document.createElement("li");
    const button = document.createElement("button");
    button.type = "button";
    button.className = "run";
    const mark = document.createElement("span");
    mark.className = "run-mark";
    mark.setAttribute("aria-hidden", "true");
    const name = document.createElement("span");
    name.className = "run-name";
    name.textContent = blockLabel(run.node);
    button.append(mark, name);
    button.addEventListener("click", () => {
      this.#setDetail(true);
      this.handlers.onRun(run.node);
    });
    item.append(button);
    const list = this.elements.runs;
    // Newest at the bottom, kept in view while the reader is there.
    const atEnd = list.scrollTop + list.clientHeight >= list.scrollHeight - 4;
    list.append(item);
    if (atEnd) list.scrollTop = list.scrollHeight;
    this.#shown.set(run.id, item);
    return item;
  }

  #open(): boolean {
    return document.body.dataset["insights"] !== "closed";
  }

  #setOpen(open: boolean, remember: boolean): void {
    document.body.dataset["insights"] = open ? "open" : "closed";
    this.elements.toggle.setAttribute("aria-pressed", String(open));
    this.elements.panel.hidden = !open;
    if (!open) this.#setDetail(false);
    if (remember) {
      try {
        localStorage.setItem(REMEMBERED, open ? "open" : "closed");
      } catch {
        // Not remembered: it opens as it did this time on the next visit.
      }
    }
    this.#nameToggle();
  }

  /** The toggle says what pressing it does. */
  #nameToggle(): void {
    const name = this.#open() ? text("insights.hide") : text("insights.show");
    this.elements.toggle.title = name;
    this.elements.toggle.setAttribute("aria-label", name);
  }

  /** On a phone, one block's details cover the screen until closed. */
  #setDetail(on: boolean): void {
    if (on) this.elements.panel.dataset["detail"] = "";
    else delete this.elements.panel.dataset["detail"];
  }
}

function remembered(): boolean | null {
  try {
    const value = localStorage.getItem(REMEMBERED);
    return value === null ? null : value === "open";
  } catch {
    return null;
  }
}
