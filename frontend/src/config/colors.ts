/**
 * Color System for ChatGameLab: the "Lab Notebook" look (#567)
 *
 * Mantine palettes have 10 shades (index 0 lightest … 9 darkest).
 *
 * The look:
 * - Paper: a warm off-white page with a faint dot pattern; cards are white.
 * - Ink: near-black text, outlines and the hard offset shadows of cards and
 *   primary buttons. Primary actions are ink.
 * - Neon cyan and magenta from the logo: accents (highlighter strokes,
 *   shadows, the active state of a control, focus), never long text.
 *
 * Readability: every text colour here is at least 4.5:1 on paper and on white.
 * Bright cyan (#29D0DE) and magenta (#FF4D9D) are below that, so they are
 * fills and lines only; text in an accent uses accent.9 or highlight.7.
 */

import type { MantineColorsTuple } from '@mantine/core';

// =============================================================================
// MANTINE COLOR TUPLES
// =============================================================================

/**
 * Ink: the primary colour. Primary buttons, checked controls, outlines, text.
 * Index 5+ are all ink, so filled controls look the same at every shade.
 */
export const inkColors: MantineColorsTuple = [
  '#F3F1EA', // 0 - hover tint on paper
  '#E6E2D6', // 1
  '#D3CEBF', // 2
  '#A8A293', // 3
  '#5B5649', // 4
  '#15140F', // 5 - ink (primaryShade)
  '#15140F', // 6
  '#0B0A08', // 7
  '#0B0A08', // 8
  '#000000', // 9
];

/**
 * Accent: neon cyan from the logo. Fills, highlighter strokes, offset
 * shadows, active markers. Text in cyan uses index 8 or 9 (≥ 5:1).
 */
export const accentColors: MantineColorsTuple = [
  '#E6FAFB', // 0 - tinted white
  '#B8F0F4', // 1
  '#8AE6ED', // 2
  '#5CDCE6', // 3
  '#3ED6E2', // 4
  '#29D0DE', // 5 - neon cyan (fill)
  '#22B8C9', // 6
  '#14808D', // 7
  '#0D6A75', // 8 - text on white, 5.9:1
  '#0B5A63', // 9 - text on white, 7.6:1
];

/**
 * Highlight: neon magenta from the logo. Shadows, underlines, the caret;
 * as text index 7 (#C81D63, 5.5:1 on white).
 */
export const highlightColors: MantineColorsTuple = [
  '#FDF2F8', // 0
  '#FCE7F3', // 1
  '#FBCFE8', // 2
  '#FF9EC8', // 3
  '#FF7EB6', // 4 - soft magenta (genre fill)
  '#FF4D9D', // 5 - neon magenta (fill, line)
  '#E3367F', // 6
  '#C81D63', // 7 - text, 5.5:1 on white
  '#9D174D', // 8
  '#831843', // 9
];

/**
 * Paper greys: warm, a little yellow, so the page reads as paper.
 * 0 page, 1 hover, 2 lines, 5 muted text (6.1:1), 7 body text, 9 ink.
 */
export const grayColors: MantineColorsTuple = [
  '#FBFAF6', // 0 - page
  '#F3F1EA', // 1 - hover / selected
  '#E6E2D6', // 2 - lines, dividers
  '#D3CEBF', // 3 - borders of quiet controls
  '#A8A293', // 4 - disabled, decorative
  '#655F52', // 5 - muted text (6.1:1 on paper)
  '#4F4A3F', // 6 - secondary text, icons
  '#3B3830', // 7 - body text
  '#24221D', // 8
  '#15140F', // 9 - titles, ink
];

/** Violet: the third neon, for the Chatbot genre and gradients. */
export const violetColors: MantineColorsTuple = [
  '#F2EFFF', // 0
  '#E2DCFF', // 1
  '#CFC6FF', // 2
  '#B9ADFF', // 3 - soft violet (genre fill)
  '#A193FF', // 4
  '#8B7BFF', // 5
  '#7462F0', // 6
  '#5B47D6', // 7 - text, 5.9:1 on white
  '#4835AD', // 8
  '#372885', // 9
];

export const successColors: MantineColorsTuple = [
  '#f0fdf4', '#dcfce7', '#bbf7d0', '#86efac', '#4ade80',
  '#22c55e', '#16a34a', '#15803d', '#166534', '#14532d',
];

export const errorColors: MantineColorsTuple = [
  '#fef2f2', '#fee2e2', '#fecaca', '#fca5a5', '#f87171',
  '#ef4444', '#dc2626', '#b91c1c', '#991b1b', '#7f1d1d',
];

export const warningColors: MantineColorsTuple = [
  '#fffbeb', '#fef3c7', '#fde68a', '#fcd34d', '#fbbf24',
  '#f59e0b', '#d97706', '#b45309', '#92400e', '#78350f',
];

export const infoColors: MantineColorsTuple = [
  '#eff6ff', '#dbeafe', '#bfdbfe', '#93c5fd', '#60a5fa',
  '#3b82f6', '#2563eb', '#1d4ed8', '#1e40af', '#1e3a8a',
];

/**
 * The page: paper with a faint dot grid, and a soft glow of cyan (top left)
 * and magenta (top right).
 */
const PAPER_BACKGROUND = [
  'radial-gradient(900px 400px at 10% -100px, rgba(41, 208, 222, 0.14), transparent 70%)',
  'radial-gradient(800px 380px at 95% -80px, rgba(255, 77, 157, 0.10), transparent 70%)',
  'radial-gradient(rgba(21, 20, 15, 0.16) 1px, transparent 1.4px) 0 0 / 22px 22px',
  grayColors[0],
].join(', ');

// =============================================================================
// SEMANTIC COLORS (theme.other.colors)
// =============================================================================

export const semanticColors = {
  // Backgrounds
  bgMain: grayColors[0],
  bgSurface: '#ffffff',
  bgHover: grayColors[1],
  bgCard: '#ffffff',
  bgCardBorder: inkColors[5],
  /** A pale yellow for a highlighted strip (a card's header, a picked option). */
  bgNote: '#FFF7B8',

  // Full-page backgrounds (landing, registration): paper with a soft glow of the two neons.
  bgLandingGradient: PAPER_BACKGROUND,
  bgRegistrationGradient: PAPER_BACKGROUND,

  // Typography
  textTitle: grayColors[9],
  textBody: grayColors[7],
  textMuted: grayColors[5],
  textInverse: '#FBFAF6',
  textAccent: accentColors[9],
  textHighlight: highlightColors[7],

  // Borders & icons
  border: grayColors[2],
  borderStrong: grayColors[3],
  ink: inkColors[5],
  iconDefault: grayColors[6],
  iconActive: inkColors[5],
} as const;

/**
 * The genres' colours: a fill each, with ink text and an ink outline on top
 * (≥ 7.8:1). Adventure cyan, Encounter magenta, Chatbot violet.
 */
export const genreColors: Record<string, string> = {
  adventure: accentColors[5],
  'npc-live': highlightColors[4],
  chatbot: violetColors[3],
};

/** A genre without its own colour. */
export const genreFallbackColor = grayColors[2];

// =============================================================================
// HEADER / DRAWER (theme.other.layout): light, on paper
// =============================================================================

export const layoutColors = {
  headerGradient: 'rgba(251, 250, 246, 0.92)',
  drawerGradient: grayColors[0],
  borderLight: `2px solid ${inkColors[5]}`,
  lineLight: grayColors[2],
  panelBg: grayColors[0],
  bgSubtle: 'transparent',
  bgHover: grayColors[1],
  bgActive: 'rgba(41, 208, 222, 0.22)',
  borderSubtle: grayColors[2],
  borderStrong: grayColors[3],
  shadowHeader: 'none',
  /** Text and icons in the header and the drawer. */
  text: inkColors[5],
  /** The active page's highlighter strip in the header. */
  activeStrip: 'linear-gradient(transparent 42%, rgba(255, 77, 157, 0.35) 42%, rgba(255, 77, 157, 0.35) 64%, transparent 64%)',
  /** Hovered nav text. */
  hoverText: highlightColors[7],
} as const;
