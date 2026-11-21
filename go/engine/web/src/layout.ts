/**
 * The pipeline drawing's layout, kept apart from the drawing so it can be
 * tested without a browser. A genre places its blocks by hand in Go; a
 * developer can drag them about on a dev server and copy the result back as Go
 * source.
 */
import type { Place } from "./client.js";

/** Where each block is drawn, by name. */
export type Layout = ReadonlyMap<string, Place>;

/**
 * The layout as the Go literal a genre keeps beside its wiring, such as
 * `adventureLayout` for "adventure". It is shifted so the topmost and leftmost
 * blocks sit at 0, rounded to whole units, ordered top to bottom and then left
 * to right, and aligned as gofmt aligns it.
 */
export function layoutCode(genre: string, layout: Layout): string {
  const places = [...layout].map(([name, p]) => ({ name, x: Math.round(p.x), y: Math.round(p.y) }));
  const top = Math.min(...places.map((p) => p.y));
  const left = Math.min(...places.map((p) => p.x));
  places.sort((a, b) => a.y - b.y || a.x - b.x || a.name.localeCompare(b.name));

  const width = Math.max(...places.map((p) => p.name.length)) + 3;
  const lines = places.map(
    (p) => `\t${`"${p.name}":`.padEnd(width)} {X: ${p.x - left}, Y: ${p.y - top}},`,
  );
  return `var ${variable(genre)} = ports.Layout{\n${lines.join("\n")}\n}\n`;
}

/** "npc-live" becomes "npcLiveLayout". */
function variable(genre: string): string {
  const words = genre.split(/[^a-zA-Z0-9]+/).filter(Boolean);
  const camel = words
    .map((w, i) => (i === 0 ? w.toLowerCase() : w[0]!.toUpperCase() + w.slice(1).toLowerCase()))
    .join("");
  return `${camel}Layout`;
}

/** How far apart parked blocks stand, and how far below the drawing. */
const PARK_STEP = 150;
const PARK_GAP = 128;

/**
 * Where blocks go that the layout misses: one row below the drawing, so a
 * block added to the wiring shows up apart and waits to be placed.
 */
export function parked(layout: Layout, unplaced: readonly string[]): Map<string, Place> {
  const ys = [...layout.values()].map((p) => p.y);
  const y = ys.length ? Math.max(...ys) + PARK_GAP : 0;
  return new Map(unplaced.map((name, i) => [name, { x: i * PARK_STEP, y }]));
}
