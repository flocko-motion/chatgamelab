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
  CardBgColor,
  FontColor,
  CardBorderThickness,
  BackgroundAnimation,
  BackgroundTint,
  PlayerIndicator,
  ThinkingStyle,
  MessageFont,
  CornerConfig,
  BackgroundConfig,
  PlayerConfig,
  GameMessageConfig,
  CardStyleConfig,
  ThinkingConfig,
  TypographyConfig,
} from './types';

// Defaults and presets
export {
  DEFAULT_GAME_THEME,
  THEME_COLORS,
  THEME_FONTS,
  PRESET_THEMES,
  CARD_BG_COLORS,
  FONT_COLORS,
  CARD_BORDER_THICKNESSES,
} from './defaults';

// Context and hook
export { GameThemeProvider } from './GameThemeContext';
export { useGameTheme, type GameThemeContextValue } from './useGameTheme';