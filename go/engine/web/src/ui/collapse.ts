/**
 * The +/− beside the pipeline's and the details' titles: each folds its pane
 * down to its title bar, and the other pane takes the room. Which pane is
 * folded is remembered, like the split between them.
 */
import { text } from "../texts.js";

type Pane = "graph" | "details";

const REMEMBERED = "engine.collapsed";

export class Collapsibles {
  readonly #folded = new Set<Pane>(remembered());

  constructor(
    private readonly insights: HTMLElement,
    private readonly toggles: Record<Pane, HTMLButtonElement>,
  ) {
    for (const pane of ["graph", "details"] as const) {
      toggles[pane].addEventListener("click", () => this.#toggle(pane));
    }
    this.#show();
  }

  /** Says the texts again, in the language now in use. */
  relabel(): void {
    for (const pane of ["graph", "details"] as const) {
      const button = this.toggles[pane];
      const label = text(this.#folded.has(pane) ? "insights.expand" : "insights.collapse");
      button.setAttribute("aria-label", label);
      button.title = label;
    }
  }

  /** Opens a pane, as picking one of its tabs does. */
  open(pane: Pane): void {
    if (!this.#folded.has(pane)) return;
    this.#toggle(pane);
  }

  #toggle(pane: Pane): void {
    if (this.#folded.has(pane)) this.#folded.delete(pane);
    else this.#folded.add(pane);
    this.#show();
    remember([...this.#folded]);
  }

  #show(): void {
    const folded = [...this.#folded];
    if (folded.length) this.insights.dataset["collapsed"] = folded.join(" ");
    else delete this.insights.dataset["collapsed"];
    for (const pane of ["graph", "details"] as const) {
      const button = this.toggles[pane];
      const open = !this.#folded.has(pane);
      button.textContent = open ? "−" : "+";
      button.setAttribute("aria-expanded", String(open));
    }
    this.relabel();
  }
}

// Storage can be missing or refuse, in a private window or a preview; both
// panes then simply start open.
function remembered(): Pane[] {
  try {
    const stored = localStorage.getItem(REMEMBERED) ?? "";
    return stored.split(" ").filter((pane): pane is Pane => pane === "graph" || pane === "details");
  } catch {
    return [];
  }
}

function remember(folded: Pane[]): void {
  try {
    if (folded.length) localStorage.setItem(REMEMBERED, folded.join(" "));
    else localStorage.removeItem(REMEMBERED);
  } catch {
    // Not remembered, which only costs a click on the next visit.
  }
}
