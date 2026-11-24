/**
 * Insights: the workings of the game beside it, for every player. The header's
 * toggle opens and closes them, remembered per browser. On a wide screen they
 * are a side panel with the pipeline and the details; on a phone a bottom pane
 * with a feed of the blocks that ran, each opening its details full screen.
 * Which of the two shows is the stylesheet's business, by the screen's width;
 * this only keeps the state both read.
 */
import type { JournalEntry } from "../client.js";
import type { PlaybackEngineState } from "../state.js";
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
  /** An entry from a block was picked in the feed: its block's details are wanted. */
  onRun(node: string): void;
  /** Whether a journal source names a block of the pipeline, which can be opened. */
  isBlock(source: string): boolean;
  /** The full-screen details were closed. */
  onClose(): void;
}

const REMEMBERED = "engine.insights";

/** Where the side panel stops and the phone's bottom pane begins (styles.css). */
const PHONE = "(max-width: 47.99rem)";

export class Insights {
  /** The journal entries drawn so far, in order: the journal only grows at its end. */
  #drawn: JournalEntry[] = [];

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
    this.#drawn = [];
    this.elements.runs.replaceChildren();
  }

  /**
   * Draws what happened: the session's journal, oldest first, newest at the
   * bottom. New entries are added at the end; a journal that was replaced
   * (a reload) or cut at its front is drawn anew.
   */
  render(state: PlaybackEngineState): void {
    const list = this.elements.runs;
    const journal = state.journal;
    const drawn = this.#drawn;
    const continues = drawn.length <= journal.length && drawn.every((entry, i) => journal[i] === entry);
    if (!continues) {
      list.replaceChildren();
      this.#drawn = [];
    }
    for (const entry of journal.slice(this.#drawn.length)) {
      // Phases change with every call; the pipeline shows them, and the
      // history keeps to what happened.
      if (entry.kind !== "phase") this.#add(entry);
      this.#drawn.push(entry);
    }
    if (journal.length === 0 && !list.querySelector(".runs-empty")) {
      const empty = document.createElement("li");
      empty.className = "runs-empty";
      empty.textContent = text("insights.empty");
      list.append(empty);
    } else if (journal.length > 0) {
      list.querySelector(".runs-empty")?.remove();
    }
  }

  #add(entry: JournalEntry): void {
    const item = document.createElement("li");
    item.dataset["level"] = entry.level;
    // An entry from a block opens that block; the session's own do not.
    const opens = this.handlers.isBlock(entry.source);
    const row = document.createElement(opens ? "button" : "div");
    if (row instanceof HTMLButtonElement) row.type = "button";
    row.className = "run";
    const mark = document.createElement("span");
    mark.className = "run-mark";
    mark.setAttribute("aria-hidden", "true");
    // When, in the reader's own time, to the second; the millisecond is a hover away.
    const when = new Date(entry.time);
    const time = document.createElement("time");
    time.className = "run-time";
    time.dateTime = entry.time;
    time.textContent = when.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
    time.title = `${time.textContent}.${String(when.getMilliseconds()).padStart(3, "0")}`;
    const name = document.createElement("span");
    name.className = "run-name";
    name.textContent = blockLabel(entry.source);
    const what = document.createElement("span");
    what.className = "run-what";
    what.textContent = entry.detail ? `${entry.kind} · ${entry.detail}` : entry.kind;
    row.append(mark, time, name, what);
    if (opens) {
      row.addEventListener("click", () => {
        this.#setDetail(true);
        this.handlers.onRun(entry.source);
      });
    }
    item.append(row);
    const list = this.elements.runs;
    // Newest at the bottom, kept in view while the reader is there.
    const atEnd = list.scrollTop + list.clientHeight >= list.scrollHeight - 4;
    list.append(item);
    if (atEnd) list.scrollTop = list.scrollHeight;
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
