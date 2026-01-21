/**
 * Default Game Theme
 * 
 * This represents the current styling as the default.
 * All customizations are optional overrides on top of this.
 */

import type { GameTheme, PartialGameTheme } from './types';

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
};


/** Card border thickness definitions */
export const CARD_BORDER_THICKNESSES: Record<string, string> = {
  none: '0',
  thin: '1px',
  medium: '2px',
  thick: '3px',
};

/** Preset themes for common game genres */
export const PRESET_THEMES: Record<string, PartialGameTheme> = {
  /** Sci-fi / Cyberpunk */
  scifi: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { tint: 'black', animation: 'stars' },
    player: { color: 'cyan', indicator: 'cursor', indicatorBlink: true, bgColor: 'cyan', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'cyan', bgColor: 'dark', fontColor: 'cyan', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Processing...', style: 'dots' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'dark', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'cyan' },
    header: { bgColor: 'black', fontColor: 'cyan', accentColor: 'cyan' },
    divider: { style: 'line', color: 'cyan' },
  },
  
  /** Fantasy / Medieval */
  fantasy: {
    corners: { style: 'flourish', color: 'amber' },
    background: { tint: 'warm', animation: 'fireflies' },
    player: { color: 'amber', indicator: 'dot', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    gameMessage: { dropCap: true, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The tale continues...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'amber', borderColor: 'amber', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'amber' },
    divider: { style: 'diamond', color: 'amber' },
  },
  
  /** Horror / Mystery */
  horror: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'dark', animation: 'rain' },
    player: { color: 'rose', indicator: 'none', indicatorBlink: false, bgColor: 'dark', fontColor: 'light', borderColor: 'rose' },
    gameMessage: { dropCap: false, dropCapColor: 'rose', bgColor: 'dark', fontColor: 'light', borderColor: 'slate' },
    cards: { borderThickness: 'none' },
    thinking: { text: 'Something stirs...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'dark', accentColor: 'rose', borderColor: 'slate', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'rose' },
    divider: { style: 'none', color: 'slate' },
  },
  
  /** Adventure / Exploration */
  adventure: {
    corners: { style: 'arrows', color: 'emerald' },
    background: { tint: 'neutral' },
    player: { color: 'emerald', indicator: 'chevron', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'emerald' },
    gameMessage: { dropCap: false, dropCapColor: 'emerald', bgColor: 'white', fontColor: 'dark', borderColor: 'emerald' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The journey continues...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'creme', accentColor: 'emerald', borderColor: 'emerald', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'dot', color: 'emerald' },
  },
  
  /** Mystery / Mystic - purple, magical, ethereal */
  mystery: {
    corners: { style: 'dots', color: 'violet' },
    background: { tint: 'darkViolet', animation: 'fireflies' },
    player: { color: 'violet', indicator: 'star', indicatorBlink: true, bgColor: 'violet', fontColor: 'light', borderColor: 'violet' },
    gameMessage: { dropCap: true, dropCapColor: 'violet', bgColor: 'dark', fontColor: 'violet', borderColor: 'violet' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'The veil thins...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'dark', accentColor: 'violet', borderColor: 'violet', fontColor: 'violet' },
    header: { bgColor: 'dark', fontColor: 'violet', accentColor: 'violet' },
    divider: { style: 'star', color: 'violet' },
  },
  
  /** Detective / Noir - grounded, stylish, classic */
  detective: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'dark' },
    player: { color: 'amber', indicator: 'pipe', indicatorBlink: false, bgColor: 'dark', fontColor: 'light', borderColor: 'amber' },
    gameMessage: { dropCap: false, dropCapColor: 'amber', bgColor: 'black', fontColor: 'light', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Investigating...', style: 'dots' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'black', accentColor: 'amber', borderColor: 'slate', fontColor: 'light' },
    header: { bgColor: 'black', fontColor: 'light', accentColor: 'amber' },
    divider: { style: 'line', color: 'slate' },
  },
  
  /** Space / Cosmic */
  space: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { tint: 'dark', animation: 'stars' },
    player: { color: 'cyan', indicator: 'dot', indicatorBlink: true, bgColor: 'dark', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'cyan', bgColor: 'dark', fontColor: 'light', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Scanning...', style: 'spinner' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'dark', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'cyan' },
    divider: { style: 'star', color: 'cyan' },
  },
  
  /** Terminal - Green on black, classic */
  terminal: {
    corners: { style: 'brackets', color: 'hacker' },
    background: { tint: 'black', animation: 'matrix' },
    player: { color: 'hacker', indicator: 'underscore', indicatorBlink: true, bgColor: 'black', fontColor: 'hacker', borderColor: 'hacker' },
    gameMessage: { dropCap: false, dropCapColor: 'hacker', bgColor: 'black', fontColor: 'hacker', borderColor: 'hacker' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Loading...', style: 'dots', streamingCursor: 'pipe' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'black', accentColor: 'hacker', borderColor: 'hacker', fontColor: 'hacker' },
    header: { bgColor: 'black', fontColor: 'hacker', accentColor: 'hacker' },
    divider: { style: 'dash', color: 'hacker' },
  },
  
  /** Default / Neutral - clean, minimal */
  default: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'neutral' },
    player: { color: 'slate', indicator: 'chevron', indicatorBlink: false, bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    gameMessage: { dropCap: false, dropCapColor: 'slate', bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The story unfolds...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'white', accentColor: 'slate', borderColor: 'slate', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'slate' },
    divider: { style: 'none', color: 'slate' },
  },
  
  /** Clean / Minimal */
  minimal: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'neutral' },
    player: { color: 'slate', indicator: 'none', indicatorBlink: false, bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    gameMessage: { dropCap: false, dropCapColor: 'slate', bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Loading...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'white', accentColor: 'slate', borderColor: 'slate', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'slate' },
    divider: { style: 'line', color: 'slate' },
  },
  
  /** Playful / Kids - Rainbow colorful theme */
  playful: {
    corners: { style: 'dots', color: 'orange' },
    background: { tint: 'blue' },
    player: { color: 'orange', indicator: 'star', indicatorBlink: true, bgColor: 'orangeLight', fontColor: 'dark', borderColor: 'orange' },
    gameMessage: { dropCap: true, dropCapColor: 'violet', bgColor: 'violetLight', fontColor: 'dark', borderColor: 'violet' },
    cards: { borderThickness: 'thick' },
    thinking: { text: 'Magic is happening...', style: 'pulse' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'cyanLight', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'dark' },
    header: { bgColor: 'greenLight', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'star', color: 'pink' },
  },
  
  /** Barbie / Pink Dream */
  barbie: {
    corners: { style: 'flourish', color: 'pink' },
    background: { tint: 'pink' },
    player: { color: 'pink', indicator: 'diamond', indicatorBlink: false, bgColor: 'pinkLight', fontColor: 'pink', borderColor: 'pink' },
    gameMessage: { dropCap: true, dropCapColor: 'pink', bgColor: 'pinkLight', fontColor: 'dark', borderColor: 'pink' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Fabulous things await...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'pinkLight', accentColor: 'pink', borderColor: 'pink', fontColor: 'pink' },
    header: { bgColor: 'pinkLight', fontColor: 'pink', accentColor: 'pink' },
    divider: { style: 'diamond', color: 'pink' },
  },
  
  /** Nature / Forest */
  nature: {
    corners: { style: 'flourish', color: 'emerald' },
    background: { tint: 'warm' },
    player: { color: 'emerald', indicator: 'chevron', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'emerald' },
    gameMessage: { dropCap: true, dropCapColor: 'emerald', bgColor: 'creme', fontColor: 'dark', borderColor: 'emerald' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The forest whispers...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'emerald', borderColor: 'emerald', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'dot', color: 'emerald' },
  },
  
  /** Ocean / Underwater */
  ocean: {
    corners: { style: 'arrows', color: 'cyan' },
    background: { tint: 'cool', animation: 'bubbles' },
    player: { color: 'cyan', indicator: 'dot', indicatorBlink: false, bgColor: 'cyanLight', fontColor: 'dark', borderColor: 'cyan' },
    gameMessage: { dropCap: true, dropCapColor: 'cyan', bgColor: 'blueLight', fontColor: 'dark', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Bubbles rise...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'blueLight', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'dark' },
    header: { bgColor: 'blueLight', fontColor: 'dark', accentColor: 'cyan' },
    divider: { style: 'dots', color: 'cyan' },
  },
  
  /** Retro / 80s */
  retro: {
    corners: { style: 'brackets', color: 'violet' },
    background: { tint: 'dark' },
    player: { color: 'cyan', indicator: 'cursor', indicatorBlink: true, bgColor: 'dark', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'violet', bgColor: 'dark', fontColor: 'light', borderColor: 'violet' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Loading...', style: 'spinner' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'dark', accentColor: 'violet', borderColor: 'violet', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'violet' },
    divider: { style: 'line', color: 'violet' },
  },
  
  /** Western / Wild West */
  western: {
    corners: { style: 'arrows', color: 'amber' },
    background: { tint: 'warm' },
    player: { color: 'amber', indicator: 'dot', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    gameMessage: { dropCap: true, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Dust settles...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'amber', borderColor: 'amber', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'amber' },
    divider: { style: 'star', color: 'amber' },
  },
  
  /** Hacker - Aggressive (Red AI, Green User) */
  hacker: {
    corners: { style: 'brackets', color: 'terminal' },
    background: { tint: 'black', animation: 'matrix' },
    player: { color: 'hacker', indicator: 'underscore', indicatorBlink: true, bgColor: 'green', fontColor: 'hacker', borderColor: 'hacker' },
    gameMessage: { dropCap: false, dropCapColor: 'terminal', bgColor: 'red', fontColor: 'terminal', borderColor: 'terminal' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'EXECUTING...', style: 'dots', streamingCursor: 'pipe' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'black', accentColor: 'terminal', borderColor: 'terminal', fontColor: 'terminal' },
    header: { bgColor: 'black', fontColor: 'terminal', accentColor: 'terminal' },
    divider: { style: 'dash', color: 'terminal' },
  },
};
