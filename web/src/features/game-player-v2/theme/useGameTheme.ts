/**
 * Hook to access the current game theme
 * 
 * Separated from context file for fast refresh compatibility.
 */

import { useContext, createContext } from 'react';
import type { GameTheme } from './types';
import { DEFAULT_GAME_THEME, THEME_COLORS, THEME_FONTS } from './defaults';

/** Merged theme with computed CSS variables */
export interface GameThemeContextValue {
  theme: GameTheme;
  /** CSS variables for the current theme */
  cssVars: Record<string, string>;
  /** Get emoji for a status field (returns empty string if not configured) */
  getStatusEmoji: (fieldName: string) => string;
}

export const GameThemeContext = createContext<GameThemeContextValue | null>(null);

/** Generate CSS variables from theme */
export function generateCssVars(theme: GameTheme): Record<string, string> {
  const cornerColor = THEME_COLORS[theme.corners.color] || THEME_COLORS.amber;
  const playerColor = THEME_COLORS[theme.player.color] || THEME_COLORS.cyan;
  const messageFont = THEME_FONTS[theme.typography.messages] || THEME_FONTS.sans;
  
  return {
    // Corner decoration colors
    '--game-corner-color': cornerColor.primary,
    '--game-corner-color-light': cornerColor.light,
    '--game-corner-color-dark': cornerColor.dark,
    
    // Player message colors
    '--game-player-color': playerColor.primary,
    '--game-player-color-light': playerColor.light,
    '--game-player-color-dark': playerColor.dark,
    '--game-player-bg': playerColor.bg,
    
    // Typography
    '--game-message-font': messageFont,
    
    // Background tint
    '--game-bg-tint': theme.background.tint === 'warm' ? 'rgba(251, 191, 36, 0.02)'
      : theme.background.tint === 'cool' ? 'rgba(34, 211, 238, 0.02)'
      : theme.background.tint === 'dark' ? 'rgba(0, 0, 0, 0.05)'
      : 'transparent',
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
      getStatusEmoji: () => '',
    };
  }
  return context;
}
