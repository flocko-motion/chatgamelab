/**
 * A "lab critter": the avatar a user gets, drawn from a seed (the user's id).
 * The same seed always gives the same critter; nothing is stored or fetched.
 * About 1,800 combinations, so neighbours in a list rarely look alike.
 */
import {
  accentColors,
  highlightColors,
  violetColors,
  successColors,
  warningColors,
  errorColors,
} from '@/config/colors';

export interface CritterParts {
  /** Body fill: a light and a main shade, and the soft background disc. */
  palette: { light: string; main: string; bg: string };
  body: number; // 0..3
  eyes: number; // 0 one, 1 two, 2 three
  mouth: number; // 0..3
  topper: number; // 0 none, 1 antenna, 2 horns, 3 sprout, 4 bubbles
  /** Where the pupils look: -1.2 (left) … 1.2 (right). */
  look: number;
}

const PALETTES: CritterParts['palette'][] = [
  { light: accentColors[2], main: accentColors[5], bg: accentColors[0] },
  { light: highlightColors[3], main: highlightColors[5], bg: highlightColors[0] },
  { light: violetColors[2], main: violetColors[5], bg: violetColors[0] },
  { light: successColors[3], main: successColors[5], bg: successColors[0] },
  { light: warningColors[3], main: warningColors[4], bg: warningColors[0] },
  { light: errorColors[3], main: errorColors[4], bg: errorColors[0] },
];

/** FNV-1a: a small, stable string hash. */
function hash(seed: string): number {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

export function critterParts(seed: string): CritterParts {
  const h = hash(seed || '?');
  const pick = (n: number, shift: number) => (h >>> shift) % n;
  return {
    palette: PALETTES[pick(PALETTES.length, 0)],
    body: pick(4, 3),
    eyes: pick(3, 6),
    mouth: pick(4, 8),
    topper: pick(5, 11),
    look: (pick(5, 14) - 2) * 0.6,
  };
}
