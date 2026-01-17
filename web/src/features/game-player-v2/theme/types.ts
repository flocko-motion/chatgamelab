/**
 * Game Player Theme Types
 * 
 * Defines the schema for customizable game player theming.
 * The current default style is preserved - themes allow non-destructive customization.
 */

/** Corner decoration styles */
export type CornerStyle = 'brackets' | 'flourish' | 'arrows' | 'dots' | 'none';

/** Available accent colors */
export type ThemeColor = 'amber' | 'emerald' | 'cyan' | 'violet' | 'rose' | 'slate';

/** Player message background colors (limited palette) */
export type PlayerBgColor = 'cyan' | 'amber' | 'violet' | 'slate' | 'white' | 'emerald' | 'rose';

/** Background animation types */
export type BackgroundAnimation = 'none' | 'stars' | 'rain' | 'fog' | 'particles' | 'scanlines';

/** Background tint */
export type BackgroundTint = 'warm' | 'cool' | 'neutral' | 'dark';

/** Player indicator icons */
export type PlayerIndicator = 'dot' | 'arrow' | 'chevron' | 'diamond' | 'none';

/** Thinking animation styles */
export type ThinkingStyle = 'dots' | 'spinner' | 'pulse' | 'typewriter';

/** Message font families */
export type MessageFont = 'serif' | 'sans' | 'mono' | 'fantasy';

/** Corner decoration configuration */
export interface CornerConfig {
  style: CornerStyle;
  color: ThemeColor;
}

/** Background configuration */
export interface BackgroundConfig {
  animation: BackgroundAnimation;
  tint: BackgroundTint;
}

/** Player message styling */
export interface PlayerConfig {
  color: ThemeColor;
  indicator: PlayerIndicator;
  monochrome: boolean;
  showChevron: boolean;
  bgColor: PlayerBgColor;
}

/** AI/Game message styling */
export interface GameMessageConfig {
  monochrome: boolean;
  dropCap: boolean;
  dropCapColor: ThemeColor;
}

/** AI thinking state configuration */
export interface ThinkingConfig {
  text: string;
  style: ThinkingStyle;
}

/** Typography configuration */
export interface TypographyConfig {
  messages: MessageFont;
}

/** Complete game theme configuration */
export interface GameTheme {
  /** Corner decoration settings */
  corners: CornerConfig;
  
  /** Background animation and tint */
  background: BackgroundConfig;
  
  /** Player input/message styling */
  player: PlayerConfig;
  
  /** AI/Game message styling */
  gameMessage: GameMessageConfig;
  
  /** AI "thinking" indicator */
  thinking: ThinkingConfig;
  
  /** Font settings */
  typography: TypographyConfig;
  
  /** Status field emoji mappings (field name -> emoji) */
  statusEmojis: Record<string, string>;
}

/** Partial theme for overrides */
export interface PartialGameTheme {
  corners?: Partial<CornerConfig>;
  background?: Partial<BackgroundConfig>;
  player?: Partial<PlayerConfig>;
  gameMessage?: Partial<GameMessageConfig>;
  thinking?: Partial<ThinkingConfig>;
  typography?: Partial<TypographyConfig>;
  statusEmojis?: Record<string, string>;
}
