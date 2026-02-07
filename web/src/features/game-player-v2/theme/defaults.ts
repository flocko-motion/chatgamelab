/**
 * Default Game Theme
 * 
 * This represents the current styling as the default.
 * All customizations are optional overrides on top of this.
 */

import type { GameTheme } from './types';

/** Default theme - clean, minimal, neutral */
export const DEFAULT_GAME_THEME: GameTheme = {
  corners: {
    style: 'none',
    color: 'slate',
    positions: {
      topLeft: true,
      topRight: false,
      bottomLeft: false,
      bottomRight: true,
    },
    blink: false,
  },
  
  background: {
    tint: 'neutral',
  },
  
  player: {
    color: 'slate',
    indicator: 'chevron',
    indicatorBlink: false,
    bgColor: 'white',
    fontColor: 'dark',
    borderColor: 'slate',
  },
  
  gameMessage: {
    dropCap: false,
    dropCapColor: 'slate',
    bgColor: 'white',
    fontColor: 'dark',
    borderColor: 'slate',
    textEffect: 'none',
  },
  
  cards: {
    borderThickness: 'thin',
  },
  
  thinking: {
    text: 'The story unfolds...',
    style: 'dots',
    streamingCursor: 'dots',
  },
  
  typography: {
    messages: 'sans',
  },
  
  statusFields: {
    bgColor: 'white',
    accentColor: 'slate',
    borderColor: 'slate',
    fontColor: 'dark',
  },
  
  header: {
    bgColor: 'white',
    fontColor: 'dark',
    accentColor: 'slate',
  },
  
  divider: {
    style: 'none',
    color: 'slate',
  },
  
  statusEmojis: {},
};

/** Color palette definitions for each theme color (used for accents: corners, chevrons, drop caps) */
export const THEME_COLORS: Record<string, { primary: string; light: string; dark: string; bg: string }> = {
  amber: {
    primary: '#d97706',
    light: '#fbbf24',
    dark: '#b45309',
    bg: 'rgba(251, 191, 36, 0.1)',
  },
  emerald: {
    primary: '#059669',
    light: '#34d399',
    dark: '#047857',
    bg: 'rgba(52, 211, 153, 0.1)',
  },
  cyan: {
    primary: '#0891b2',
    light: '#22d3ee',
    dark: '#0e7490',
    bg: 'rgba(34, 211, 238, 0.1)',
  },
  violet: {
    primary: '#7c3aed',
    light: '#a78bfa',
    dark: '#6d28d9',
    bg: 'rgba(167, 139, 250, 0.1)',
  },
  rose: {
    primary: '#e11d48',
    light: '#fb7185',
    dark: '#be123c',
    bg: 'rgba(251, 113, 133, 0.1)',
  },
  slate: {
    primary: '#475569',
    light: '#94a3b8',
    dark: '#334155',
    bg: 'rgba(148, 163, 184, 0.1)',
  },
  // Classic hacker/terminal green
  hacker: {
    primary: '#00ff00',
    light: '#39ff14',
    dark: '#00cc00',
    bg: 'rgba(0, 255, 0, 0.1)',
  },
  // Terminal red
  terminal: {
    primary: '#ff0000',
    light: '#ff4444',
    dark: '#cc0000',
    bg: 'rgba(255, 0, 0, 0.1)',
  },
  // Brown (dark/earthy)
  brown: {
    primary: '#78350f',
    light: '#a16207',
    dark: '#451a03',
    bg: 'rgba(120, 53, 15, 0.1)',
  },
  // Brown Light (warm tan)
  brownLight: {
    primary: '#d97706',
    light: '#fbbf24',
    dark: '#92400e',
    bg: 'rgba(217, 119, 6, 0.1)',
  },
  // Pink (vibrant)
  pink: {
    primary: '#db2777',
    light: '#f472b6',
    dark: '#9d174d',
    bg: 'rgba(219, 39, 119, 0.1)',
  },
  // Pink Light (soft/pastel)
  pinkLight: {
    primary: '#ec4899',
    light: '#f9a8d4',
    dark: '#be185d',
    bg: 'rgba(236, 72, 153, 0.1)',
  },
  // Orange (vibrant)
  orange: {
    primary: '#ea580c',
    light: '#fb923c',
    dark: '#c2410c',
    bg: 'rgba(234, 88, 12, 0.1)',
  },
  // Orange Light (soft/warm)
  orangeLight: {
    primary: '#f97316',
    light: '#fdba74',
    dark: '#ea580c',
    bg: 'rgba(249, 115, 22, 0.1)',
  },
  // Sky blue (friendly, school)
  sky: {
    primary: '#0284c7',
    light: '#38bdf8',
    dark: '#0369a1',
    bg: 'rgba(56, 189, 248, 0.1)',
  },
  // Indigo (deep blue-violet)
  indigo: {
    primary: '#4f46e5',
    light: '#818cf8',
    dark: '#4338ca',
    bg: 'rgba(129, 140, 248, 0.1)',
  },
  // Lime (bright green)
  lime: {
    primary: '#65a30d',
    light: '#a3e635',
    dark: '#4d7c0f',
    bg: 'rgba(163, 230, 53, 0.1)',
  },
  // Sunshine (warm yellow)
  sunshine: {
    primary: '#ca8a04',
    light: '#facc15',
    dark: '#a16207',
    bg: 'rgba(250, 204, 21, 0.1)',
  },
  // Coral (warm pink-orange)
  coral: {
    primary: '#f43f5e',
    light: '#fb7185',
    dark: '#e11d48',
    bg: 'rgba(251, 113, 133, 0.1)',
  },
  // Lavender (soft purple)
  lavender: {
    primary: '#a78bfa',
    light: '#c4b5fd',
    dark: '#8b5cf6',
    bg: 'rgba(196, 181, 253, 0.1)',
  },
  // Teal (blue-green)
  teal: {
    primary: '#0d9488',
    light: '#2dd4bf',
    dark: '#0f766e',
    bg: 'rgba(45, 212, 191, 0.1)',
  },
};

/** Font family mappings */
export const THEME_FONTS: Record<string, string> = {
  serif: 'Georgia, "Times New Roman", serif',
  sans: 'Inter, system-ui, -apple-system, sans-serif',
  mono: '"Fira Code", "JetBrains Mono", Consolas, monospace',
  fantasy: '"Cinzel", "Palatino Linotype", serif',
};

/** Card background color definitions */
export const CARD_BG_COLORS: Record<string, { solid: string; alpha: string }> = {
  // Neutrals
  white: { solid: '#ffffff', alpha: 'rgba(255, 255, 255, 0.95)' },
  creme: { solid: '#fdf8f3', alpha: 'rgba(253, 248, 243, 0.95)' },
  dark: { solid: '#1e1e2e', alpha: 'rgba(30, 30, 46, 0.95)' },
  black: { solid: '#0a0a0a', alpha: 'rgba(10, 10, 10, 0.95)' },
  // Blue
  blue: { solid: '#1e3a5f', alpha: 'rgba(30, 58, 95, 0.95)' },
  blueLight: { solid: '#e0f2fe', alpha: 'rgba(224, 242, 254, 0.95)' },
  // Green
  green: { solid: '#001a00', alpha: 'rgba(0, 26, 0, 0.95)' },
  greenLight: { solid: '#dcfce7', alpha: 'rgba(220, 252, 231, 0.95)' },
  // Red
  red: { solid: '#1a0000', alpha: 'rgba(26, 0, 0, 0.95)' },
  redLight: { solid: '#fee2e2', alpha: 'rgba(254, 226, 226, 0.95)' },
  // Amber
  amber: { solid: '#78350f', alpha: 'rgba(120, 53, 15, 0.95)' },
  amberLight: { solid: '#fef3c7', alpha: 'rgba(254, 243, 199, 0.95)' },
  // Violet
  violet: { solid: '#2e1065', alpha: 'rgba(46, 16, 101, 0.95)' },
  violetLight: { solid: '#ede9fe', alpha: 'rgba(237, 233, 254, 0.95)' },
  // Rose
  rose: { solid: '#4c0519', alpha: 'rgba(76, 5, 25, 0.95)' },
  roseLight: { solid: '#ffe4e6', alpha: 'rgba(255, 228, 230, 0.95)' },
  // Cyan
  cyan: { solid: '#164e63', alpha: 'rgba(22, 78, 99, 0.95)' },
  cyanLight: { solid: '#cffafe', alpha: 'rgba(207, 250, 254, 0.95)' },
  // Pink
  pink: { solid: '#831843', alpha: 'rgba(131, 24, 67, 0.95)' },
  pinkLight: { solid: '#fce7f3', alpha: 'rgba(252, 231, 243, 0.95)' },
  // Orange
  orange: { solid: '#7c2d12', alpha: 'rgba(124, 45, 18, 0.95)' },
  orangeLight: { solid: '#ffedd5', alpha: 'rgba(255, 237, 213, 0.95)' },
  // Sky
  skyLight: { solid: '#e0f2fe', alpha: 'rgba(224, 242, 254, 0.95)' },
  // Indigo
  indigoLight: { solid: '#e0e7ff', alpha: 'rgba(224, 231, 255, 0.95)' },
  // Lime
  limeLight: { solid: '#ecfccb', alpha: 'rgba(236, 252, 203, 0.95)' },
  // Sunshine / Yellow
  sunshineLight: { solid: '#fef9c3', alpha: 'rgba(254, 249, 195, 0.95)' },
  // Coral
  coralLight: { solid: '#ffe4e6', alpha: 'rgba(255, 228, 230, 0.95)' },
  // Lavender
  lavenderLight: { solid: '#ede9fe', alpha: 'rgba(237, 233, 254, 0.95)' },
  // Teal
  tealLight: { solid: '#ccfbf1', alpha: 'rgba(204, 251, 241, 0.95)' },
};

/** Font color definitions */
export const FONT_COLORS: Record<string, string> = {
  dark: '#1a1a2e',
  light: '#f5f5f5',
  hacker: '#00ff00',
  terminal: '#ff0000',
  pink: '#ec4899',
  amber: '#f59e0b',
  cyan: '#06b6d4',
  violet: '#8b5cf6',
  sky: '#0284c7',
  indigo: '#4f46e5',
  lime: '#65a30d',
  sunshine: '#ca8a04',
  coral: '#f43f5e',
  lavender: '#a78bfa',
  teal: '#0d9488',
};


/** Card border thickness definitions */
export const CARD_BORDER_THICKNESSES: Record<string, string> = {
  none: '0',
  thin: '1px',
  medium: '2px',
  thick: '3px',
};

