/** Sine-wave bands scrolling sideways along the bottom third of the scene. */
import { backdrop, div, svg } from "./dom.js";

type WaveBand = readonly [
  amplitude: number,
  frequency: number,
  /** In viewBox units, of a height of 100. */
  yOffset: number,
  stroke: string,
  fill: string,
  fillOpacity: number,
  strokeOpacity: number,
  duration: string,
  strokeWidth: number,
];

const bands: readonly WaveBand[] = [
  [6, 2, 15, "#3b82f6", "#3b82f6", 0.06, 0.3, "14s", 1.5],
  [4, 3, 35, "#60a5fa", "#60a5fa", 0.05, 0.25, "11s", 1],
  [8, 1.5, 50, "#2563eb", "#2563eb", 0.08, 0.2, "16s", 2],
  [5, 2.5, 65, "#93c5fd", "#93c5fd", 0.04, 0.2, "12s", 1],
  [3, 4, 80, "#60a5fa", "#60a5fa", 0.03, 0.15, "9s", 0.8],
];

const svgWidth = 800;
const svgHeight = 100;

export function mountWaves(layer: HTMLElement): void {
  const root = backdrop("bg-waves");
  for (const [amplitude, frequency, yOffset, stroke, fill, fillOpacity, strokeOpacity, duration, strokeWidth] of bands) {
    const band = div("bg-wave");
    band.style.animationDuration = duration;
    const picture = svg("svg", { viewBox: `0 0 ${svgWidth * 2} ${svgHeight}`, preserveAspectRatio: "none" });
    picture.append(
      svg("path", {
        d: filledWave(svgWidth * 2, amplitude, frequency * 2, yOffset, svgHeight),
        fill,
        "fill-opacity": fillOpacity,
        stroke,
        "stroke-opacity": strokeOpacity,
        "stroke-width": strokeWidth,
      }),
    );
    band.append(picture);
    root.append(band);
  }
  layer.append(root);
}

/** A closed path: a sine wave along the top, filled down to the bottom edge. */
function filledWave(width: number, amplitude: number, frequency: number, yOffset: number, height: number): string {
  const points: string[] = [];
  const steps = 200;
  for (let i = 0; i <= steps; i++) {
    const x = (i / steps) * width;
    const y = yOffset + Math.sin((i / steps) * Math.PI * 2 * frequency) * amplitude;
    points.push(`${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`);
  }
  points.push(`L${width},${height} L0,${height} Z`);
  return points.join(" ");
}
