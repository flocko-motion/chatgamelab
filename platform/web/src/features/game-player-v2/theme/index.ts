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
  TextEffect,
  TextEffectScope,
} from "./types";

// Defaults
export {
  DEFAULT_GAME_THEME,
  THEME_COLORS,
  THEME_FONTS,
  CARD_BG_COLORS,
  FONT_COLORS,
  CARD_BORDER_THICKNESSES,
} from "./defaults";

// Presets
export { PRESETS } from "./presets";
export type { PresetDefinition, MessageTextWrapperProps } from "./presets";

// Context and hook
export { GameThemeProvider } from "./GameThemeContext";
export { useGameTheme, type GameThemeContextValue } from "./useGameTheme";
