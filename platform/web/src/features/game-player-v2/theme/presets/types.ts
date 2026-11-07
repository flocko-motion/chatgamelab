/**
 * Preset Definition Type
 *
 * Defines the shape of a theme preset. Each preset provides a theme config
 * and can optionally include custom React components for advanced visuals.
 *
 * This interface is designed to be extended as new customization points are needed.
 * Adding a new optional field here does not break existing presets.
 */

import type { ComponentType, ReactNode } from "react";
import type { PartialGameTheme } from "../types";

/** Props for message text wrapper components */
export interface MessageTextWrapperProps {
  children: ReactNode;
  text: string;
  isStreaming?: boolean;
}

export interface PresetDefinition {
  /** The theme config (colors, corners, typography, etc.) */
  theme: PartialGameTheme;

  /** Optional custom background component (replaces tsparticles animation) */
  BackgroundComponent?: ComponentType<{ className?: string }>;

  /** Optional wrapper for AI/game message text (completed messages) */
  GameMessageWrapper?: ComponentType<MessageTextWrapperProps>;

  /** Optional wrapper for player message text */
  PlayerMessageWrapper?: ComponentType<MessageTextWrapperProps>;

  /** Optional wrapper for AI message text while streaming */
  StreamingMessageWrapper?: ComponentType<MessageTextWrapperProps>;
}
