/**
 * Hook to access the current game theme
 *
 * Separated from context file for fast refresh compatibility.
 */

import { useContext, createContext, type ComponentType } from "react";
import type { GameTheme } from "./types";
import {
  DEFAULT_GAME_THEME,
  THEME_COLORS,
  THEME_FONTS,
  CARD_BG_COLORS,
  FONT_COLORS,
  CARD_BORDER_THICKNESSES,
} from "./defaults";

/** Merged theme with computed CSS variables */
export interface GameThemeContextValue {
  theme: GameTheme;
  /** CSS variables for the current theme */
  cssVars: Record<string, string>;
  /** Get emoji for a status field (returns empty string if not configured) */
  getStatusEmoji: (fieldName: string) => string;
  /** Optional custom background component from preset (replaces tsparticles) */
  BackgroundComponent?: ComponentType<{ className?: string }>;
}

export const GameThemeContext = createContext<GameThemeContextValue | null>(
  null,
);

/** Generate CSS variables from theme */
export function generateCssVars(theme: GameTheme): Record<string, string> {
  const cornerColor = THEME_COLORS[theme.corners.color] || THEME_COLORS.amber;
  const playerColor = THEME_COLORS[theme.player.color] || THEME_COLORS.cyan;
  const dropCapColor =
    THEME_COLORS[theme.gameMessage.dropCapColor] || THEME_COLORS.amber;
  const messageFont =
    THEME_FONTS[theme.typography.messages] || THEME_FONTS.sans;

  // Card backgrounds
  const playerBgColor =
    CARD_BG_COLORS[theme.player.bgColor] || CARD_BG_COLORS.white;
  const gameBgColor =
    CARD_BG_COLORS[theme.gameMessage.bgColor] || CARD_BG_COLORS.white;

  // Font colors
  const playerFontColor =
    FONT_COLORS[theme.player.fontColor] || FONT_COLORS.dark;
  const gameFontColor =
    FONT_COLORS[theme.gameMessage.fontColor] || FONT_COLORS.dark;

  // Card styling (shared)
  const borderWidth =
    CARD_BORDER_THICKNESSES[theme.cards.borderThickness] ||
    CARD_BORDER_THICKNESSES.thin;

  // Per-message border colors (using ThemeColor)
  const playerBorderColor =
    THEME_COLORS[theme.player.borderColor] || THEME_COLORS.cyan;
  const gameBorderColor =
    THEME_COLORS[theme.gameMessage.borderColor] || THEME_COLORS.amber;

  // Image highlight color (always uses AI border color)

  // Status field colors (with safe fallbacks for undefined statusFields)
  const statusBgColor =
    CARD_BG_COLORS[theme.statusFields?.bgColor] || CARD_BG_COLORS.creme;
  const statusAccentColor =
    THEME_COLORS[theme.statusFields?.accentColor] || THEME_COLORS.amber;
  const statusBorderColor =
    THEME_COLORS[theme.statusFields?.borderColor] || THEME_COLORS.amber;
  const statusFontColor =
    FONT_COLORS[theme.statusFields?.fontColor] || FONT_COLORS.dark;

  // Header colors (with safe fallbacks)
  const headerBgColor =
    CARD_BG_COLORS[theme.header?.bgColor] || CARD_BG_COLORS.white;
  const headerFontColor =
    FONT_COLORS[theme.header?.fontColor] || FONT_COLORS.dark;
  const headerAccentColor =
    THEME_COLORS[theme.header?.accentColor] || THEME_COLORS.amber;

  // Divider colors (with safe fallbacks)
  const dividerColor = THEME_COLORS[theme.divider?.color] || THEME_COLORS.amber;

  // Error colors - adapt to dark vs light card backgrounds
  const isDarkPlayer = [
    "dark",
    "black",
    "blue",
    "green",
    "red",
    "amber",
    "violet",
    "rose",
    "cyan",
    "pink",
    "orange",
  ].includes(theme.player.bgColor);
  const errorBg = isDarkPlayer
    ? "rgba(251, 113, 133, 0.15)"
    : "rgba(239, 68, 68, 0.1)";
  const errorBorder = isDarkPlayer
    ? "rgba(251, 113, 133, 0.4)"
    : "rgba(239, 68, 68, 0.4)";
  const errorText = isDarkPlayer ? "#fb7185" : "#b91c1c";

  return {
    // Corner decoration colors
    "--game-corner-color": cornerColor.primary,
    "--game-corner-color-light": cornerColor.light,
    "--game-corner-color-dark": cornerColor.dark,

    // Player message styling
    "--game-player-color": playerColor.primary,
    "--game-player-color-light": playerColor.light,
    "--game-player-color-dark": playerColor.dark,
    "--game-player-bg": playerBgColor.solid,
    "--game-player-font-color": playerFontColor,

    // Game/AI message styling
    "--game-ai-bg": gameBgColor.solid,
    "--game-ai-font-color": gameFontColor,

    // Drop cap color
    "--game-drop-cap-color": dropCapColor.primary,

    // Card styling (shared)
    "--game-border-width": borderWidth,

    // Per-message border colors
    "--game-player-border-color": playerBorderColor.primary,
    "--game-ai-border-color": gameBorderColor.primary,

    // Typography
    "--game-message-font": messageFont,

    // Background tint
    "--game-bg-tint":
      theme.background.tint === "warm"
        ? "rgba(251, 191, 36, 0.08)"
        : theme.background.tint === "cool"
          ? "rgba(34, 211, 238, 0.08)"
          : theme.background.tint === "dark"
            ? "#0f0f1a"
            : theme.background.tint === "black"
              ? "#000000"
              : theme.background.tint === "pink"
                ? "rgba(236, 72, 153, 0.12)"
                : theme.background.tint === "green"
                  ? "rgba(34, 197, 94, 0.10)"
                  : theme.background.tint === "blue"
                    ? "rgba(59, 130, 246, 0.10)"
                    : theme.background.tint === "violet"
                      ? "rgba(139, 92, 246, 0.10)"
                      : theme.background.tint === "darkCyan"
                        ? "#0a1a1f"
                        : theme.background.tint === "darkViolet"
                          ? "#1a0f2e"
                          : theme.background.tint === "darkBlue"
                            ? "#0a0f1a"
                            : theme.background.tint === "darkRose"
                              ? "#1a0a0f"
                              : "transparent",

    // Image highlight color (uses AI border color)
    "--game-image-highlight": gameBorderColor.primary,
    "--game-image-highlight-bg": gameBorderColor.bg,

    // Status field colors
    "--game-status-bg": statusBgColor.solid,
    "--game-status-accent": statusAccentColor.primary,
    "--game-status-border": statusBorderColor.primary,
    "--game-status-font": statusFontColor,

    // Header colors
    "--game-header-bg": headerBgColor.solid,
    "--game-header-font": headerFontColor,
    "--game-header-accent": headerAccentColor.primary,

    // Divider colors
    "--game-divider-color": dividerColor.primary,

    // Error colors (theme-adaptive)
    "--game-error-bg": errorBg,
    "--game-error-border": errorBorder,
    "--game-error-text": errorText,
  };
}

/** Hook to access the current game theme */
export function useGameTheme(): GameThemeContextValue {
  const context = useContext(GameThemeContext);
  if (!context) {
    // Return defaults if used outside provider (graceful fallback)
    const theme = DEFAULT_GAME_THEME;
    return {
      theme,
      cssVars: generateCssVars(theme),
      getStatusEmoji: () => "",
    };
  }
  return context;
}
