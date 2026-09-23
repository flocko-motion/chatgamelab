/** A glowing orb arcing slowly across the sky. */
import { backdrop, div } from "./dom.js";

export function mountSun(layer: HTMLElement): void {
  const root = backdrop();
  root.append(div("bg-sun"));
  layer.append(root);
}
