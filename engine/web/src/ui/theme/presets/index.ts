import type { PresetDefinition } from "./types.js";
import { SIMPLE_PRESETS } from "./simple.js";
import { ANIMATED_PRESETS } from "./animated.js";

/** Every preset, keyed by the name the theme generator picks. */
export const PRESETS: Record<string, PresetDefinition> = {
  ...SIMPLE_PRESETS,
  ...ANIMATED_PRESETS,
};

export type { PresetDefinition } from "./types.js";
