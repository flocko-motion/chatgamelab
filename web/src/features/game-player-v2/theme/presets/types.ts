/**
 * Preset Definition Type
 *
 * Defines the shape of a theme preset. Each preset provides a theme config
 * and can optionally include custom React components for advanced visuals.
 *
 * This interface is designed to be extended as new customization points are needed.
 * Adding a new optional field here does not break existing presets.
 */

import type { ComponentType } from 'react';
import type { PartialGameTheme } from '../types';

export interface PresetDefinition {
  /** The theme config (colors, corners, typography, etc.) */
  theme: PartialGameTheme;

  /** Optional custom background component (replaces tsparticles animation) */
  BackgroundComponent?: ComponentType<{ className?: string }>;

  // Future extension points:
  // CursorComponent?: ComponentType<{ className?: string }>;
  // CardWrapper?: ComponentType<{ children: ReactNode; className?: string }>;
  // DividerComponent?: ComponentType<{ className?: string }>;
}
