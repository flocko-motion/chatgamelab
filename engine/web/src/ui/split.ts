/**
 * A draggable divider between the game and the instrumentation.
 *
 * The width is one custom property on <body>, so dragging writes a value and
 * the grid reflows; nothing here measures or positions a column itself. The
 * choice is remembered, because how much room the graph deserves is a matter of
 * what someone is doing — playing, or watching how the play is assembled — and
 * they should not have to say so again on every reload.
 */
const REMEMBERED = "engine.split";

/** Neither half is allowed to shrink to a sliver it cannot be dragged back from. */
const MIN_PANE = 280;

export function mountSplit(divider: HTMLElement): void {
  const remembered = Number(localStorage.getItem(REMEMBERED));
  if (Number.isFinite(remembered) && remembered > 0) apply(remembered);

  divider.addEventListener("pointerdown", (event: PointerEvent) => {
    event.preventDefault();
    divider.setPointerCapture(event.pointerId);
    divider.dataset["dragging"] = "";
    document.body.dataset["dragging"] = "";
  });

  divider.addEventListener("pointermove", (event: PointerEvent) => {
    if (!divider.hasPointerCapture(event.pointerId)) return;
    apply(event.clientX);
  });

  const finish = (event: PointerEvent): void => {
    if (!divider.hasPointerCapture(event.pointerId)) return;
    divider.releasePointerCapture(event.pointerId);
    delete divider.dataset["dragging"];
    delete document.body.dataset["dragging"];
    localStorage.setItem(REMEMBERED, String(clamp(event.clientX)));
  };
  divider.addEventListener("pointerup", finish);
  divider.addEventListener("pointercancel", finish);

  // A double click restores the even split the page opens with, which is the
  // cheapest way back from a drag that went too far.
  divider.addEventListener("dblclick", () => {
    document.body.style.removeProperty("--split");
    localStorage.removeItem(REMEMBERED);
  });
}

function apply(x: number): void {
  document.body.style.setProperty("--split", `${clamp(x)}px`);
}

function clamp(x: number): number {
  const widest = Math.max(MIN_PANE, window.innerWidth - MIN_PANE);
  return Math.min(Math.max(x, MIN_PANE), widest);
}
