/**
 * Game Theme Provider Component
 *
 * Provides theme configuration to all game player components.
 * Falls back to defaults for any missing values.
 */

import { useMemo, type ReactNode, type ComponentType } from "react";
import type { MessageTextWrapperProps } from "./presets/types";
import type { GameTheme, PartialGameTheme } from "./types";
import { DEFAULT_GAME_THEME } from "./defaults";
import {
  GameThemeContext,
  generateCssVars,
  type GameThemeContextValue,
} from "./useGameTheme";

/** Deep merge partial theme with defaults */
function mergeTheme(partial: PartialGameTheme | undefined): GameTheme {
  if (!partial) return DEFAULT_GAME_THEME;

  return {
    corners: {
      ...DEFAULT_GAME_THEME.corners,
      ...partial.corners,
    } as GameTheme["corners"],
    background: {
      ...DEFAULT_GAME_THEME.background,
      ...partial.background,
    } as GameTheme["background"],
    player: {
      ...DEFAULT_GAME_THEME.player,
      ...partial.player,
    } as GameTheme["player"],
    gameMessage: {
      ...DEFAULT_GAME_THEME.gameMessage,
      ...partial.gameMessage,
      textEffectScope: {
        ...DEFAULT_GAME_THEME.gameMessage.textEffectScope,
        ...partial.gameMessage?.textEffectScope,
      },
    } as GameTheme["gameMessage"],
    cards: {
      ...DEFAULT_GAME_THEME.cards,
      ...partial.cards,
    } as GameTheme["cards"],
    thinking: {
      ...DEFAULT_GAME_THEME.thinking,
      ...partial.thinking,
    } as GameTheme["thinking"],
    typography: {
      ...DEFAULT_GAME_THEME.typography,
      ...partial.typography,
    } as GameTheme["typography"],
    statusFields: {
      ...DEFAULT_GAME_THEME.statusFields,
      ...partial.statusFields,
    } as GameTheme["statusFields"],
    header: {
      ...DEFAULT_GAME_THEME.header,
      ...partial.header,
    } as GameTheme["header"],
    divider: {
      ...DEFAULT_GAME_THEME.divider,
      ...partial.divider,
    } as GameTheme["divider"],
    statusEmojis: {
      ...DEFAULT_GAME_THEME.statusEmojis,
      ...(partial.statusEmojis || {}),
    },
  };
}

interface GameThemeProviderProps {
  children: ReactNode;
  /** Partial theme to merge with defaults */
  theme?: PartialGameTheme;
  /** Optional custom background component from preset */
  BackgroundComponent?: ComponentType<{ className?: string }>;
  /** Optional wrapper for AI/game message text (completed messages) */
  GameMessageWrapper?: ComponentType<MessageTextWrapperProps>;
  /** Optional wrapper for player message text */
  PlayerMessageWrapper?: ComponentType<MessageTextWrapperProps>;
  /** Optional wrapper for AI message text while streaming */
  StreamingMessageWrapper?: ComponentType<MessageTextWrapperProps>;
}

export function GameThemeProvider({
  children,
  theme: partialTheme,
  BackgroundComponent,
  GameMessageWrapper,
  PlayerMessageWrapper,
  StreamingMessageWrapper,
}: GameThemeProviderProps) {
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
      return "";
    };

    return {
      theme,
      cssVars,
      getStatusEmoji,
      BackgroundComponent,
      GameMessageWrapper,
      PlayerMessageWrapper,
      StreamingMessageWrapper,
    };
  }, [
    partialTheme,
    BackgroundComponent,
    GameMessageWrapper,
    PlayerMessageWrapper,
    StreamingMessageWrapper,
  ]);

  return (
    <GameThemeContext.Provider value={value}>
      {children}
    </GameThemeContext.Provider>
  );
}
