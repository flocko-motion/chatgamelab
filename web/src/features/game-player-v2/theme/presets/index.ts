/**
 * Preset Registry
 *
 * Collects all preset definitions into a single registry.
 * - Simple (config-only) presets live in simple.ts
 * - Complex presets with custom components get their own folder
 */

import type { PresetDefinition } from './types';
import { SIMPLE_PRESETS } from './simple';
import { ANIMATED_PRESETS } from './animated';

/** All available presets, keyed by name */
export const PRESETS: Record<string, PresetDefinition> = {
  ...SIMPLE_PRESETS,
  ...ANIMATED_PRESETS,
  // Custom-component presets override here, e.g.:
  // space: spacePreset,  (from ./space/)
};

export type { PresetDefinition } from './types';
