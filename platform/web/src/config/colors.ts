/**
 * Color System for ChatGameLab
 * 
 * This file defines color palettes in Mantine's 10-shade format (indices 0-9).
 * 
 * Brand Identity:
 * - Dark bluish gradient header → trustworthy, techy, calm
 * - Neon cyan/magenta logo → playful, game-y, creative
 * - Light theme content → modern SaaS, readable, scalable
 *
 * Usage Rules:
 * - Cyan (accent) → primary actions, highlights, focus states
 * - Magenta (highlight) → attention, notifications, special actions (use sparingly!)
 * - Everything else → calm, neutral, predictable
 * - "If everything is colorful, nothing is important"
 * 
 * Mantine Index Reference:
 * - Index 0-2: Light shades (backgrounds, subtle states)
 * - Index 3-4: Medium-light (hover states)
 * - Index 5-6: Main color (default, filled buttons) ← primaryShade
 * - Index 7-8: Dark shades (pressed states, borders)
 * - Index 9: Darkest (text on light bg, high contrast)
 */

import type { MantineColorsTuple } from '@mantine/core';

// =============================================================================
// MANTINE COLOR TUPLES (10 shades each, index 0-9)
// =============================================================================

/**
 * Primary Accent - Cyan (main: #29D0DE)
 * Use for: Primary buttons, active nav, toggles, focus rings, links
 */
export const accentColors: MantineColorsTuple = [
  '#e6fafb', // 0 - lightest (tinted white)
  '#b8f0f4', // 1
  '#8ae6ed', // 2
  '#5cdce6', // 3
  '#3ed6e2', // 4
  '#29D0DE', // 5 - main accent (primaryShade)
  '#22b8c9', // 6 - hover/darker
  '#1a9baa', // 7
  '#137e8b', // 8
  '#0d616c', // 9 - darkest (for text)
];

/**
 * Secondary Accent - Magenta (from logo #FF4D9D)  
 * Use for: Notifications, badges, special actions (Create, New, Pro)
 * ⚠️ Don't overuse - one accent per screen is ideal
 */
export const highlightColors: MantineColorsTuple = [
  '#fdf2f8', // 0
  '#fce7f3', // 1
  '#fbcfe8', // 2
  '#f9a8d4', // 3
  '#f472b6', // 4
  '#FF4D9D', // 5 - main highlight
  '#db2777', // 6
  '#be185d', // 7
  '#9d174d', // 8
  '#831843', // 9
];

/**
 * Slate-based neutrals (slightly blue-tinted for cohesion with header)
 */
export const grayColors: MantineColorsTuple = [
  '#F6F8FB', // 0 - main app background
  '#EEF2F7', // 1 - hover/selected backgrounds
  '#E2E8F0', // 2 - borders, dividers
  '#cbd5e1', // 3
  '#94a3b8', // 4
  '#64748B', // 5 - muted text
  '#475569', // 6 - icon default
  '#334155', // 7 - body text
  '#1e293b', // 8
  '#0F172A', // 9 - title text (dark blue, not pure black)
];

/**
 * Success - Green
 */
export const successColors: MantineColorsTuple = [
  '#f0fdf4', // 0
  '#dcfce7', // 1
  '#bbf7d0', // 2
  '#86efac', // 3
  '#4ade80', // 4
  '#22c55e', // 5
  '#16a34a', // 6
  '#15803d', // 7
  '#166534', // 8
  '#14532d', // 9
];

/**
 * Error - Red
 */
export const errorColors: MantineColorsTuple = [
  '#fef2f2', // 0
  '#fee2e2', // 1
  '#fecaca', // 2
  '#fca5a5', // 3
  '#f87171', // 4
  '#ef4444', // 5
  '#dc2626', // 6
  '#b91c1c', // 7
  '#991b1b', // 8
  '#7f1d1d', // 9
];

/**
 * Warning - Orange/Amber
 */
export const warningColors: MantineColorsTuple = [
  '#fffbeb', // 0
  '#fef3c7', // 1
  '#fde68a', // 2
  '#fcd34d', // 3
  '#fbbf24', // 4
  '#f59e0b', // 5
  '#d97706', // 6
  '#b45309', // 7
  '#92400e', // 8
  '#78350f', // 9
];

/**
 * Info - Blue (using Mantine's default blue, slightly adjusted)
 */
export const infoColors: MantineColorsTuple = [
  '#eff6ff', // 0
  '#dbeafe', // 1
  '#bfdbfe', // 2
  '#93c5fd', // 3
  '#60a5fa', // 4
  '#3b82f6', // 5
  '#2563eb', // 6
  '#1d4ed8', // 7
  '#1e40af', // 8
  '#1e3a8a', // 9
];

// =============================================================================
// SEMANTIC COLORS (for direct use in components)
// =============================================================================

export const semanticColors = {
  // Backgrounds
  bgMain: grayColors[0],        // #F6F8FB - main app background
  bgSurface: '#ffffff',         // cards, modals, dropdowns
  bgHover: grayColors[1],       // #EEF2F7 - hover/selected states
  bgCard: '#f0f4f8',            // bluish card bg (matches header tone)
  bgCardBorder: '#d1dce8',      // bluish card border
  
  // Gradients
  bgLandingGradient: 'linear-gradient(180deg, #f0f4f8 0%, #e8eef5 25%, #dfe8f2 50%, #e8eef5 75%, #f0f4f8 100%)',
  bgRegistrationGradient: 'linear-gradient(180deg, #f0f4f8 0%, #e8eef5 25%, #dfe8f2 50%, #e8eef5 75%, #f0f4f8 100%)',
  
  // Typography
  textTitle: grayColors[9],     // #0F172A - headlines
  textBody: grayColors[7],      // #334155 - body text
  textMuted: grayColors[5],     // #64748B - secondary text
  textInverse: '#E6F0FF',       // text on dark backgrounds
  
  // Borders & Icons
  border: grayColors[2],        // #E2E8F0 - subtle borders
  borderStrong: grayColors[3],  // #cbd5e1 - emphasized borders
  iconDefault: grayColors[6],   // #475569 - default icon color
  iconActive: accentColors[5],  // cyan for active icons
} as const;

// =============================================================================
// HEADER/DARK THEME LAYOUT COLORS
// =============================================================================

export const layoutColors = {
  headerGradient: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)',
  drawerGradient: 'linear-gradient(180deg, #1a1a2e 0%, #16213e 100%)',
  borderLight: '1px solid rgba(255, 255, 255, 0.1)',
  lineLight: 'rgba(255, 255, 255, 0.1)',
  panelBg: '#1a1a2e',
  bgSubtle: 'rgba(15, 52, 96, 0.25)',
  bgHover: 'rgba(15, 52, 96, 0.45)',
  bgActive: 'rgba(15, 52, 96, 0.6)',
  borderSubtle: 'rgba(255, 255, 255, 0.18)',
  borderStrong: 'rgba(255, 255, 255, 0.26)',
  shadowHeader: '0 2px 10px rgba(0, 0, 0, 0.3)',
} as const;
