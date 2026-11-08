/**
 * Tumbleweeds bouncing along the bottom, for the western theme. Each weed runs
 * a single transform animation combining drift, bounce and spin, which stays
 * on the compositor.
 */
import { backdrop, div, svg } from "./dom.js";

interface Weed {
  readonly size: number;
  readonly bottom: number;
  readonly opacity: number;
  /** One of the `bg-tw*` keyframes in backgrounds.css. */
  readonly keyframes: string;
  readonly duration: string;
  readonly delay: string;
  readonly fill: string;
}

const weeds: readonly Weed[] = [
  { size: 70, bottom: 2, opacity: 0.35, keyframes: "bg-tw1", duration: "12s", delay: "0s", fill: "#92400e" },
  { size: 25, bottom: 5, opacity: 0.22, keyframes: "bg-tw2", duration: "7s", delay: "3s", fill: "#a16207" },
  { size: 85, bottom: 1, opacity: 0.3, keyframes: "bg-tw3", duration: "18s", delay: "8s", fill: "#78350f" },
  { size: 20, bottom: 7, opacity: 0.2, keyframes: "bg-tw4", duration: "5s", delay: "1s", fill: "#b45309" },
  { size: 45, bottom: 3, opacity: 0.25, keyframes: "bg-tw1", duration: "20s", delay: "14s", fill: "#92400e" },
  { size: 30, bottom: 4, opacity: 0.22, keyframes: "bg-tw3", duration: "9s", delay: "18s", fill: "#a16207" },
  { size: 95, bottom: 1, opacity: 0.28, keyframes: "bg-tw2", duration: "24s", delay: "22s", fill: "#78350f" },
];

export function mountTumbleweed(layer: HTMLElement): void {
  const root = backdrop();
  for (const weed of weeds) {
    const node = div("bg-tumbleweed");
    node.style.bottom = `${weed.bottom}%`;
    node.style.width = `${weed.size}px`;
    node.style.height = `${weed.size}px`;
    node.style.opacity = String(weed.opacity);
    node.style.animation = `${weed.keyframes} ${weed.duration} linear ${weed.delay} infinite`;
    node.append(bush(weed.fill));
    root.append(node);
  }
  layer.append(root);
}

/** Overlapping wobbly ellipses, for a tangled look. */
function bush(fill: string): SVGElement {
  const picture = svg("svg", { viewBox: "0 0 40 40" });
  picture.append(
    svg("ellipse", { cx: 20, cy: 20, rx: 16, ry: 14, fill, "fill-opacity": 0.15, stroke: "#78350f", "stroke-width": 1.2, opacity: 0.6 }),
    svg("ellipse", { cx: 18, cy: 19, rx: 13, ry: 15, fill: "none", stroke: "#92400e", "stroke-width": 1.5, opacity: 0.5 }),
    svg("ellipse", { cx: 22, cy: 21, rx: 14, ry: 12, fill: "none", stroke: "#a16207", "stroke-width": 1.3, opacity: 0.5 }),
    svg("ellipse", { cx: 20, cy: 18, rx: 11, ry: 13, fill: "none", stroke: "#92400e", "stroke-width": 1, opacity: 0.4 }),
    svg("ellipse", { cx: 19, cy: 22, rx: 12, ry: 10, fill: "none", stroke: "#b45309", "stroke-width": 1.2, opacity: 0.4 }),
    svg("circle", { cx: 21, cy: 20, r: 8, fill: "none", stroke: "#78350f", "stroke-width": 0.8, opacity: 0.3 }),
    svg("circle", { cx: 19, cy: 20, r: 6, fill: "none", stroke: "#92400e", "stroke-width": 0.8, opacity: 0.3 }),
    svg("line", { x1: 10, y1: 16, x2: 30, y2: 24, stroke: "#92400e", "stroke-width": 0.8, opacity: 0.3 }),
    svg("line", { x1: 12, y1: 26, x2: 28, y2: 14, stroke: "#a16207", "stroke-width": 0.8, opacity: 0.25 }),
    svg("line", { x1: 14, y1: 12, x2: 26, y2: 28, stroke: "#78350f", "stroke-width": 0.7, opacity: 0.2 }),
  );
  return picture;
}
