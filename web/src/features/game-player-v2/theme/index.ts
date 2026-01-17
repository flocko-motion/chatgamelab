/**
 * Game Theme System
 * 
 * Exports for customizable game player theming.
 */

// Types
export type {
  GameTheme,
  PartialGameTheme,
  CornerStyle,
  ThemeColor,
  PlayerBgColor,
  BackgroundAnimation,
  BackgroundTint,
  PlayerIndicator,
  ThinkingStyle,
  MessageFont,
  CornerConfig,
  BackgroundConfig,
  PlayerConfig,
  GameMessageConfig,
  ThinkingConfig,
  TypographyConfig,
} from './types';

// Defaults and presets
export {
  DEFAULT_GAME_THEME,
  THEME_COLORS,
  THEME_FONTS,
  PRESET_THEMES,
} from './defaults';

// Context and hook
export { GameThemeProvider } from './GameThemeContext';
export { useGameTheme, type GameThemeContextValue } from './useGameTheme';