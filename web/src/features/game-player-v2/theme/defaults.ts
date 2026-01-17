/**
 * Default Game Theme
 * 
 * This represents the current styling as the default.
 * All customizations are optional overrides on top of this.
 */

import type { GameTheme } from './types';

/** Default theme - matches current sci-fi/tech aesthetic */
export const DEFAULT_GAME_THEME: GameTheme = {
  corners: {
    style: 'brackets',
    color: 'amber',
  },
  
  background: {
    animation: 'none',
    tint: 'warm',
  },
  
  player: {
    color: 'cyan',
    indicator: 'dot',
    monochrome: true,
    showChevron: true,
    bgColor: 'cyan',
  },
  
  gameMessage: {
    monochrome: false,
    dropCap: false,
    dropCapColor: 'amber',
  },
  
  thinking: {
    text: 'The story unfolds...',
    style: 'dots',
  },
  
  typography: {
    messages: 'sans',
  },
  
  statusEmojis: {},
};

/** Color palette definitions for each theme color */
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
};

/** Font family mappings */
export const THEME_FONTS: Record<string, string> = {
  serif: 'Georgia, "Times New Roman", serif',
  sans: 'Inter, system-ui, -apple-system, sans-serif',
  mono: '"Fira Code", "JetBrains Mono", Consolas, monospace',
  fantasy: '"Cinzel", "Palatino Linotype", serif',
};

/** Preset themes for common game genres */
export const PRESET_THEMES: Record<string, Partial<GameTheme>> = {
  /** Sci-fi / Cyberpunk - current default */
  scifi: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { animation: 'scanlines', tint: 'cool' },
    player: { color: 'cyan', indicator: 'dot', monochrome: true, showChevron: true, bgColor: 'cyan' },
    gameMessage: { monochrome: false, dropCap: false, dropCapColor: 'cyan' },
    thinking: { text: 'Processing...', style: 'dots' },
    typography: { messages: 'mono' },
  },
  
  /** Fantasy / Medieval */
  fantasy: {
    corners: { style: 'flourish', color: 'amber' },
    background: { animation: 'particles', tint: 'warm' },
    player: { color: 'amber', indicator: 'diamond', monochrome: false, showChevron: false, bgColor: 'amber' },
    gameMessage: { monochrome: false, dropCap: true, dropCapColor: 'amber' },
    thinking: { text: 'The tale continues...', style: 'typewriter' },
    typography: { messages: 'serif' },
  },
  
  /** Horror / Mystery */
  horror: {
    corners: { style: 'none', color: 'slate' },
    background: { animation: 'fog', tint: 'dark' },
    player: { color: 'rose', indicator: 'none', monochrome: true, showChevron: false, bgColor: 'slate' },
    gameMessage: { monochrome: true, dropCap: false, dropCapColor: 'rose' },
    thinking: { text: 'Something stirs...', style: 'pulse' },
    typography: { messages: 'serif' },
  },
  
  /** Adventure / Exploration */
  adventure: {
    corners: { style: 'arrows', color: 'emerald' },
    background: { animation: 'particles', tint: 'neutral' },
    player: { color: 'emerald', indicator: 'arrow', monochrome: false, showChevron: true, bgColor: 'emerald' },
    gameMessage: { monochrome: false, dropCap: false, dropCapColor: 'emerald' },
    thinking: { text: 'The journey continues...', style: 'dots' },
    typography: { messages: 'sans' },
  },
  
  /** Mystery / Detective */
  mystery: {
    corners: { style: 'dots', color: 'violet' },
    background: { animation: 'rain', tint: 'cool' },
    player: { color: 'violet', indicator: 'chevron', monochrome: false, showChevron: true, bgColor: 'violet' },
    gameMessage: { monochrome: false, dropCap: false, dropCapColor: 'violet' },
    thinking: { text: 'Investigating...', style: 'spinner' },
    typography: { messages: 'sans' },
  },
  
  /** Space / Cosmic */
  space: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { animation: 'stars', tint: 'dark' },
    player: { color: 'cyan', indicator: 'dot', monochrome: true, showChevron: true, bgColor: 'slate' },
    gameMessage: { monochrome: false, dropCap: false, dropCapColor: 'cyan' },
    thinking: { text: 'Scanning...', style: 'spinner' },
    typography: { messages: 'mono' },
  },
};
