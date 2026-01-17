/**
 * Game Theme Provider Component
 * 
 * Provides theme configuration to all game player components.
 * Falls back to defaults for any missing values.
 */

import { useMemo, type ReactNode } from 'react';
import type { GameTheme, PartialGameTheme } from './types';
import { DEFAULT_GAME_THEME } from './defaults';
import { GameThemeContext, generateCssVars, type GameThemeContextValue } from './useGameTheme';

/** Deep merge partial theme with defaults */
function mergeTheme(partial: PartialGameTheme | undefined): GameTheme {
  if (!partial) return DEFAULT_GAME_THEME;
  
  return {
    corners: { ...DEFAULT_GAME_THEME.corners, ...partial.corners } as GameTheme['corners'],
    background: { ...DEFAULT_GAME_THEME.background, ...partial.background } as GameTheme['background'],
    player: { ...DEFAULT_GAME_THEME.player, ...partial.player } as GameTheme['player'],
    gameMessage: { ...DEFAULT_GAME_THEME.gameMessage, ...partial.gameMessage } as GameTheme['gameMessage'],
    thinking: { ...DEFAULT_GAME_THEME.thinking, ...partial.thinking } as GameTheme['thinking'],
    typography: { ...DEFAULT_GAME_THEME.typography, ...partial.typography } as GameTheme['typography'],
    statusEmojis: { ...DEFAULT_GAME_THEME.statusEmojis, ...(partial.statusEmojis || {}) },
  };
}

interface GameThemeProviderProps {
  children: ReactNode;
  /** Partial theme to merge with defaults */
  theme?: PartialGameTheme;
}

export function GameThemeProvider({ children, theme: partialTheme }: GameThemeProviderProps) {
  const value = useMemo<GameThemeContextValue>(() => {
    const theme = mergeTheme(partialTheme);
    const cssVars = generateCssVars(theme);
    
    const getStatusEmoji = (fieldName: string): string => {
      // Check exact match first
      if (theme.statusEmojis[fieldName]) {
        return theme.statusEmojis[fieldName];
      }
      // Check case-insensitive match
      const lowerName = fieldName.toLowerCase();
      for (const [key, emoji] of Object.entries(theme.statusEmojis)) {
        if (key.toLowerCase() === lowerName) {
          return emoji;
        }
      }
      return '';
    };
    
    return { theme, cssVars, getStatusEmoji };
  }, [partialTheme]);
  
  return (
    <GameThemeContext.Provider value={value}>
      {children}
    </GameThemeContext.Provider>
  );
}
