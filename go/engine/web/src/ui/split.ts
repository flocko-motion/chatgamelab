/**
 * Draggable dividers: one between the game and the insights, one inside the
 * insights between the pipeline and the details.
 *
 * Each size is one custom property, so dragging writes a value and the layout
 * reflows; nothing here positions a pane itself. The choice is remembered,
 * because how much room the graph deserves is a matter of what someone is
 * doing — playing, or watching how the play is assembled — and they should not
 * have to say so again on every reload.
 */

/** One divider: what it resizes, along which axis, and where the size lives. */
interface Split {
  divider: HTMLElement;
  /** "x" drags sideways, "y" up and down. */
  axis: "x" | "y";
  /** The element holding the custom property. */
  holder: HTMLElement;
  property: string;
  remembered: string;
  /** The pane the size is measured from: its leading edge is the origin. */
  pane: HTMLElement;
  /** The room both panes share, and the least either may keep. */
  room: () => number;
  minPane: number;
}

/** Neither half is allowed to shrink to a sliver it cannot be dragged back from. */
const MIN_COLUMN = 280;
const MIN_ROW = 160;

/** The divider between the game and the insights. */
export function mountSplit(divider: HTMLElement): void {
  mount({
    divider,
    axis: "x",
    holder: document.body,
    property: "--split",
    remembered: "engine.split",
    pane: document.body,
    room: () => window.innerWidth,
    minPane: MIN_COLUMN,
  });
}

/** The divider between the pipeline and the details below it. */
export function mountInsightsSplit(
  divider: HTMLElement,
  insights: HTMLElement,
  graph: HTMLElement,
  details: HTMLElement,
): void {
  mount({
    divider,
    axis: "y",
    holder: insights,
    property: "--graph-height",
    remembered: "engine.insightsSplit",
    pane: graph,
    // The graph and the details share the column's height.
    room: () => graph.offsetHeight + details.offsetHeight,
    minPane: MIN_ROW,
  });
}

function mount(split: Split): void {
  const { divider } = split;
  const stored = read(split.remembered);
  if (stored > 0) apply(split, stored);

  divider.addEventListener("pointerdown", (event: PointerEvent) => {
    event.preventDefault();
    divider.setPointerCapture(event.pointerId);
    divider.dataset["dragging"] = "";
    document.body.dataset["dragging"] = split.axis;
  });

  divider.addEventListener("pointermove", (event: PointerEvent) => {
    if (!divider.hasPointerCapture(event.pointerId)) return;
    apply(split, offset(split, event));
  });

  const finish = (event: PointerEvent): void => {
    if (!divider.hasPointerCapture(event.pointerId)) return;
    divider.releasePointerCapture(event.pointerId);
    delete divider.dataset["dragging"];
    delete document.body.dataset["dragging"];
    write(split.remembered, String(clamp(split, offset(split, event))));
  };
  divider.addEventListener("pointerup", finish);
  divider.addEventListener("pointercancel", finish);

  // A double click restores the split the page opens with, which is the
  // cheapest way back from a drag that went too far.
  divider.addEventListener("dblclick", () => {
    split.holder.style.removeProperty(split.property);
    write(split.remembered, null);
  });
}

/** Where the pointer is, measured from the pane's leading edge. */
function offset(split: Split, event: PointerEvent): number {
  const box = split.pane.getBoundingClientRect();
  return split.axis === "x" ? event.clientX - box.left : event.clientY - box.top;
}

function apply(split: Split, size: number): void {
  split.holder.style.setProperty(split.property, `${clamp(split, size)}px`);
}

function clamp(split: Split, size: number): number {
  const widest = Math.max(split.minPane, split.room() - split.minPane);
  return Math.min(Math.max(size, split.minPane), widest);
}

// Storage can be missing or refuse, in a private window or a preview; the
// split then simply starts even.
function read(key: string): number {
  try {
    const value = Number(localStorage.getItem(key));
    return Number.isFinite(value) ? value : 0;
  } catch {
    return 0;
  }
}

function write(key: string, value: string | null): void {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    // Not remembered, which only costs a drag on the next visit.
  }
}
