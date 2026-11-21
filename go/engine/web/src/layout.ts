/**
 * The pipeline drawing's grid, kept apart from the drawing so it can be tested
 * without a browser. A genre places its blocks by hand in Go; a developer can
 * rearrange them on a dev server and copy the result back as Go source.
 */
import type { Cell } from "./client.js";

/** The drawing's places, by block name. */
export type Layout = ReadonlyMap<string, Cell>;

/**
 * The layout as the Go literal a genre keeps beside its wiring, such as
 * `adventureLayout` for "adventure". It is shifted to start at row 0, column
 * 0, ordered by row and then column, and aligned as gofmt aligns it, so
 * pasting it over the old one leaves a diff of only what moved.
 */
export function layoutCode(genre: string, layout: Layout): string {
  const cells = [...layout].map(([name, cell]) => ({ name, ...cell }));
  const top = Math.min(...cells.map((c) => c.row));
  const left = Math.min(...cells.map((c) => c.col));
  cells.sort((a, b) => a.row - b.row || a.col - b.col || a.name.localeCompare(b.name));

  const width = Math.max(...cells.map((c) => c.name.length)) + 3;
  const lines = cells.map(
    (c) => `\t${`"${c.name}":`.padEnd(width)} {Row: ${c.row - top}, Col: ${c.col - left}},`,
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

/**
 * Where blocks go that the layout misses: one row, two below the lowest placed
 * one, so a block added to the wiring shows up apart and waits to be placed.
 */
export function parked(layout: Layout, unplaced: readonly string[]): Map<string, Cell> {
  const rows = [...layout.values()].map((c) => c.row);
  const row = rows.length ? Math.max(...rows) + 2 : 0;
  return new Map(unplaced.map((name, col) => [name, { row, col }]));
}
