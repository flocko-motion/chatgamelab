/**
 * The problems tab: the warnings and errors the blocks and the session
 * reported, from the session's journal, grouped by who reported them, newest
 * first. A source that is a block opens that block's details.
 */
import type { JournalEntry } from "../client.js";
import { blockLabel, text } from "../texts.js";

type ProblemEntry = JournalEntry;

export class Problems {
  /** What was last drawn: the journal grows in place, so its length and last entry tell. */
  #shown: { journal: readonly ProblemEntry[]; length: number; last: ProblemEntry | undefined } | null = null;

  constructor(
    private readonly root: HTMLElement,
    private readonly onSource: (source: string) => void,
    /** Whether a source names a block of the pipeline, which can be opened. */
    private readonly isBlock: (source: string) => boolean,
  ) {
    this.render([]);
  }

  /** The number of problems, for the tab's badge. */
  static count(journal: readonly ProblemEntry[]): number {
    return journal.filter(isProblem).length;
  }

  /** Repaints when the journal changed; an unchanged one is left alone. */
  render(journal: readonly ProblemEntry[]): void {
    const last = journal[journal.length - 1];
    const shown = this.#shown;
    if (shown && shown.journal === journal && shown.length === journal.length && shown.last === last) return;
    this.#shown = { journal, length: journal.length, last };

    const problems = journal.filter(isProblem);
    if (problems.length === 0) {
      this.root.replaceChildren(line("p", "meta", text("insights.noProblems")));
      return;
    }

    // Grouped by source, the group with the newest problem first.
    const groups = new Map<string, ProblemEntry[]>();
    for (const entry of [...problems].reverse()) {
      const group = groups.get(entry.source) ?? [];
      group.push(entry);
      groups.set(entry.source, group);
    }

    const sections: HTMLElement[] = [];
    for (const [source, entries] of groups) {
      const section = document.createElement("section");
      section.className = "problem-group";
      const head = document.createElement("h4");
      if (this.isBlock(source)) {
        const open = document.createElement("button");
        open.type = "button";
        open.className = "problem-source";
        open.textContent = blockLabel(source);
        open.addEventListener("click", () => this.onSource(source));
        head.append(open);
      } else {
        head.textContent = blockLabel(source);
      }
      head.append(line("span", "problem-count", String(entries.length)));
      section.append(head);
      for (const entry of entries) section.append(row(entry));
      sections.push(section);
    }
    this.root.replaceChildren(...sections);
  }
}

function isProblem(entry: ProblemEntry): boolean {
  return entry.level === "warning" || entry.level === "error";
}

function row(entry: ProblemEntry): HTMLElement {
  const item = document.createElement("div");
  item.className = "problem";
  item.dataset["level"] = entry.level;
  item.append(
    line("span", "problem-meta", [clock(entry.time), entry.level, entry.kind].filter(Boolean).join(" · ")),
    line("span", "problem-detail", entry.detail ?? ""),
  );
  return item;
}

/** Time of day to the millisecond: enough to order what arrives together. */
function clock(at: string): string {
  const when = new Date(at);
  if (Number.isNaN(when.getTime())) return "";
  return when.toLocaleTimeString(undefined, { hour12: false }) +
    "." + String(when.getMilliseconds()).padStart(3, "0");
}

function line(tag: string, className: string, content: string): HTMLElement {
  const element = document.createElement(tag);
  if (className) element.className = className;
  element.textContent = content;
  return element;
}
