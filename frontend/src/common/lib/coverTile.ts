/**
 * The tile a game shows while it has no cover: a gradient of two theme
 * colours at an angle, both picked from the game's id. The same game always
 * gets the same tile, a renamed one keeps it, and neighbours in a list
 * differ.
 */

/** The theme's palettes a tile draws from: each a Mantine colour. */
const PALETTES = ["accent", "highlight", "blue", "violet", "indigo", "teal", "grape", "green", "orange"];

export interface CoverTile {
  /** A CSS gradient in theme colours. */
  background: string;
}

/** FNV-1a: a small, stable string hash. */
function hash(seed: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** The tile for a game, by its id. */
export function coverTile(seed: string): CoverTile {
  const h = hash(seed);
  const n = PALETTES.length;
  const from = h % n;
  // A second colour, never the first.
  const to = (from + 1 + ((h >>> 8) % (n - 1))) % n;
  const angle = 90 + ((h >>> 16) % 180);
  return {
    background: `linear-gradient(${angle}deg, var(--mantine-color-${PALETTES[from]}-6) 0%, var(--mantine-color-${PALETTES[to]}-8) 100%)`,
  };
}
