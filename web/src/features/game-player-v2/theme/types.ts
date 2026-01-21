/**
 * Game Player Theme Types
 * 
 * Defines the schema for customizable game player theming.
 * The current default style is preserved - themes allow non-destructive customization.
 */

/** Corner decoration styles */
export type CornerStyle = 'brackets' | 'flourish' | 'arrows' | 'dots' | 'dot' | 'cursor' | 'none';

/** Available accent colors (corners, chevrons, drop caps, player accents) */
export type ThemeColor = 
  | 'amber' | 'emerald' | 'cyan' | 'violet' | 'rose' | 'slate' 
  | 'hacker' | 'terminal'
  | 'brown' | 'brownLight' | 'pink' | 'pinkLight' | 'orange' | 'orangeLight';

/** Message card background colors */
export type CardBgColor = 
  | 'white' | 'creme' | 'dark' | 'black' 
  | 'blue' | 'blueLight' | 'green' | 'greenLight' | 'red' | 'redLight'
  | 'amber' | 'amberLight' | 'violet' | 'violetLight' | 'rose' | 'roseLight' | 'cyan' | 'cyanLight'
  | 'pink' | 'pinkLight' | 'orange' | 'orangeLight';

/** Font colors for messages */
export type FontColor = 'dark' | 'light' | 'hacker' | 'terminal' | 'pink' | 'amber' | 'cyan' | 'violet';

/** Card border thickness */
export type CardBorderThickness = 'none' | 'thin' | 'medium' | 'thick';


/** Background animation types */
export type BackgroundAnimation = 'none' | 'stars' | 'bubbles' | 'fireflies' | 'snow' | 'rain' | 'matrix';

/** Background tint */
export type BackgroundTint = 
  | 'warm' | 'cool' | 'neutral' | 'dark' | 'black'
  | 'pink' | 'green' | 'blue' | 'violet'
  | 'darkCyan' | 'darkViolet' | 'darkBlue' | 'darkRose';

/** Player indicator icons */
export type PlayerIndicator = 'dot' | 'chevron' | 'pipe' | 'cursor' | 'underscore' | 'diamond' | 'arrow' | 'star' | 'none';

/** Thinking animation styles */
export type ThinkingStyle = 'dots' | 'spinner' | 'pulse' | 'typewriter';

/** Streaming cursor styles (shown while text is being generated) */
export type StreamingCursor = 'dots' | 'block' | 'pipe' | 'underscore' | 'none';

/** Message font families */
export type MessageFont = 'serif' | 'sans' | 'mono' | 'fantasy';

/** Corner position configuration */
export interface CornerPositions {
  topLeft: boolean;
  topRight: boolean;
  bottomLeft: boolean;
  bottomRight: boolean;
}

/** Corner decoration configuration */
export interface CornerConfig {
  style: CornerStyle;
  color: ThemeColor;
  positions?: CornerPositions;
  blink?: boolean;
}

/** Background configuration */
export interface BackgroundConfig {
  tint: BackgroundTint;
  animation?: BackgroundAnimation;
}

/** Player message styling */
export interface PlayerConfig {
  color: ThemeColor;
  indicator: PlayerIndicator;
  indicatorBlink: boolean;
  bgColor: CardBgColor;
  fontColor: FontColor;
  borderColor: ThemeColor;
}

/** AI/Game message styling */
export interface GameMessageConfig {
  dropCap: boolean;
  dropCapColor: ThemeColor;
  bgColor: CardBgColor;
  fontColor: FontColor;
  borderColor: ThemeColor;
}

/** Status field styling */
export interface StatusFieldConfig {
  bgColor: CardBgColor;
  accentColor: ThemeColor;
  borderColor: ThemeColor;
  fontColor: FontColor;
}

/** Header bar styling */
export interface HeaderConfig {
  bgColor: CardBgColor;
  fontColor: FontColor;
  accentColor: ThemeColor;
}

/** Divider style between messages */
export type DividerStyle = 'dot' | 'line' | 'dots' | 'diamond' | 'star' | 'dash' | 'none';

/** Divider configuration */
export interface DividerConfig {
  style: DividerStyle;
  color: ThemeColor;
}

/** Card styling shared between player and game messages */
export interface CardStyleConfig {
  borderThickness: CardBorderThickness;
}

/** AI thinking state configuration */
export interface ThinkingConfig {
  text: string;
  style: ThinkingStyle;
  /** Cursor shown while streaming text */
  streamingCursor: StreamingCursor;
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
  
  /** Shared card styling (transparency) */
  cards: CardStyleConfig;
  
  /** AI "thinking" indicator */
  thinking: ThinkingConfig;
  
  /** Font settings */
  typography: TypographyConfig;
  
  /** Status field styling */
  statusFields: StatusFieldConfig;
  
  /** Header bar styling */
  header: HeaderConfig;
  
  /** Divider between messages */
  divider: DividerConfig;
  
  /** Status field emoji mappings (field name -> emoji) */
  statusEmojis: Record<string, string>;
}

/** Partial theme for overrides */
export interface PartialGameTheme {
  corners?: Partial<CornerConfig>;
  background?: Partial<BackgroundConfig>;
  player?: Partial<PlayerConfig>;
  gameMessage?: Partial<GameMessageConfig>;
  cards?: Partial<CardStyleConfig>;
  thinking?: Partial<ThinkingConfig>;
  typography?: Partial<TypographyConfig>;
  statusFields?: Partial<StatusFieldConfig>;
  header?: Partial<HeaderConfig>;
  divider?: Partial<DividerConfig>;
  statusEmojis?: Record<string, string>;
}
