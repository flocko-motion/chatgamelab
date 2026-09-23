import type { PartialGameTheme } from "../types.js";

/** A named theme. The platform's presets carry nothing but configuration. */
export interface PresetDefinition {
  theme: PartialGameTheme;
}
