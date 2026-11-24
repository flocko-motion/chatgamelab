/**
 * The tabs of the insights' lower pane: the session's usage, its guardrail,
 * what is selected, what happened, and the problems blocks reported. Which one is open
 * is remembered; picking a block opens its tab, the background goes back to
 * usage. Which panel shows is the stylesheet's business, by the pane's
 * data-tab; on a phone there are no tabs, only the feed.
 */
import { text, type TextKey } from "../texts.js";

export type Tab = "block" | "usage" | "guardrail" | "feed" | "problems";

// Usage first: it is what the pane shows when nothing is selected. The
// guardrail beside it: like usage, it belongs to the whole session.
const TABS: readonly Tab[] = ["usage", "guardrail", "block", "feed", "problems"];

const LABELS: Record<Tab, TextKey> = {
  block: "insights.tabSelected",
  usage: "insights.tabUsage",
  guardrail: "insights.tabGuardrail",
  feed: "insights.feed",
  problems: "insights.tabProblems",
};

const REMEMBERED = "engine.insightsTab";

export class InsightsTabs {
  #current: Tab;
  readonly #buttons = new Map<Tab, HTMLButtonElement>();
  readonly #counts = new Map<Tab, HTMLElement>();

  constructor(
    private readonly pane: HTMLElement,
    strip: HTMLElement,
    /** A tab was picked, by the reader or because a block was. */
    private readonly onSelect: (tab: Tab) => void = () => {},
  ) {
    strip.setAttribute("role", "tablist");
    for (const tab of TABS) {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "insights-tab";
      button.id = `tab-${tab}`;
      button.setAttribute("role", "tab");
      button.setAttribute("aria-controls", `panel-${tab}`);
      const name = document.createElement("span");
      const count = document.createElement("span");
      count.className = "tab-count";
      count.hidden = true;
      button.append(name, count);
      button.addEventListener("click", () => this.select(tab));
      // Arrow keys move between the tabs, as a tab list does.
      button.addEventListener("keydown", (event) => {
        const step = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
        if (!step) return;
        event.preventDefault();
        const next = TABS[(TABS.indexOf(tab) + step + TABS.length) % TABS.length]!;
        this.select(next);
        this.#buttons.get(next)?.focus();
      });
      strip.append(button);
      this.#buttons.set(tab, button);
      this.#counts.set(tab, count);
    }
    this.#current = remembered() ?? "usage";
    this.#show();
    this.relabel();
  }

  get current(): Tab {
    return this.#current;
  }

  select(tab: Tab): void {
    this.onSelect(tab);
    if (tab === this.#current) return;
    this.#current = tab;
    this.#show();
    try {
      localStorage.setItem(REMEMBERED, tab);
    } catch {
      // Not remembered: the next visit opens on usage.
    }
  }

  /** A number on a tab, such as how many problems; zero shows none. */
  count(tab: Tab, n: number): void {
    const badge = this.#counts.get(tab);
    if (!badge) return;
    badge.hidden = n === 0;
    badge.textContent = n > 99 ? "99+" : String(n);
  }

  /** Says the texts again, in the language now in use. */
  relabel(): void {
    for (const [tab, button] of this.#buttons) {
      const name = button.firstElementChild as HTMLElement;
      name.textContent = text(LABELS[tab]);
    }
  }

  #show(): void {
    this.pane.dataset["tab"] = this.#current;
    for (const [tab, button] of this.#buttons) {
      const on = tab === this.#current;
      button.setAttribute("aria-selected", String(on));
      button.tabIndex = on ? 0 : -1;
    }
  }
}

function remembered(): Tab | null {
  try {
    const stored = localStorage.getItem(REMEMBERED);
    return TABS.includes(stored as Tab) ? (stored as Tab) : null;
  } catch {
    return null;
  }
}
