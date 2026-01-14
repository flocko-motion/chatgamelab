// Color palette configuration for ChatGameLab
// Centralized color definitions - all color decisions live here
//
// Brand Identity:
// - Dark bluish gradient header → trustworthy, techy, calm
// - Neon cyan/magenta logo → playful, game-y, creative
// - Light theme content → modern SaaS, readable, scalable
//
// Usage Rules:
// - Cyan → primary actions, highlights, focus states
// - Magenta → attention, notifications, special actions
// - Everything else → calm, neutral, predictable
// - If everything is colorful, nothing is important

export const colors = {
  // ==========================================================================
  // BRAND ACCENT COLORS (from logo)
  // ==========================================================================
  
  // Primary Accent - Cyan (logo cyan #22E6F3)
  // Use for: Primary buttons, active nav, toggles, sliders, focus rings
  accent: {
    50: '#ecfeff',
    100: '#cffafe',
    200: '#a5f3fc',
    300: '#67e8f9',
    400: '#22d3ee',
    500: '#2ADDEC', // Main accent - cyan
    600: '#28D5E4', // Hover state (subtle darkening)
    700: '#0e7490',
    800: '#155e75',
    900: '#164e63',
    950: '#083344',
  },
  
  // Secondary Accent - Magenta (logo magenta #FF4D9D)
  // Use for: Notifications, badges, special actions (Create, New, Pro)
  // ⚠️ Don't overuse - one accent per screen is ideal
  highlight: {
    50: '#fdf2f8',
    100: '#fce7f3',
    200: '#fbcfe8',
    300: '#f9a8d4',
    400: '#f472b6',
    500: '#FF4D9D', // Main highlight - logo magenta
    600: '#db2777',
    700: '#be185d',
    800: '#9d174d',
    900: '#831843',
    950: '#500724',
  },

  // ==========================================================================
  // SEMANTIC COLORS
  // ==========================================================================
  
  success: {
    50: '#f0fdf4',
    100: '#dcfce7',
    200: '#bbf7d0',
    300: '#86efac',
    400: '#4ade80',
    500: '#22c55e',
    600: '#16a34a',
    700: '#15803d',
    800: '#166534',
    900: '#14532d',
    950: '#052e16',
  },

  error: {
    50: '#fef2f2',
    100: '#fee2e2',
    200: '#fecaca',
    300: '#fca5a5',
    400: '#f87171',
    500: '#ef4444',
    600: '#dc2626',
    700: '#b91c1c',
    800: '#991b1b',
    900: '#7f1d1d',
    950: '#450a0a',
  },

  warning: {
    50: '#fffbeb',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b',
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
    950: '#451a03',
  },

  // ==========================================================================
  // NEUTRAL / SURFACE COLORS
  // ==========================================================================
  
  // Slate-based neutrals (slightly blue-tinted for cohesion with header)
  gray: {
    50: '#F6F8FB',  // Main app background - soft, reduces eye strain
    100: '#EEF2F7', // Hover/selected backgrounds
    200: '#E2E8F0', // Borders, dividers
    300: '#cbd5e1',
    400: '#94a3b8',
    500: '#64748B', // Muted text
    600: '#475569', // Icon default
    700: '#334155', // Body text
    800: '#1e293b',
    900: '#0F172A', // Title text - dark blue instead of pure black
    950: '#020617',
  },

  // Special colors
  white: '#ffffff',
  black: '#000000',
  transparent: 'transparent',
} as const;

// ==========================================================================
// SEMANTIC COLOR ROLES
// ==========================================================================

export const semanticColors = {
  // Accent colors (use sparingly)
  accentPrimary: colors.accent[500],      // #2ADDEC - cyan
  accentSecondary: colors.highlight[500], // #FF4D9D - magenta
  
  // Backgrounds
  bgMain: colors.gray[50],      // #F6F8FB - main app background
  bgSurface: colors.white,      // #FFFFFF - cards, modals, dropdowns
  bgHover: colors.gray[100],    // #EEF2F7 - hover/selected states
  bgCard: '#f0f4f8',            // Bluish card background (matches header tone)
  bgCardBorder: '#d1dce8',      // Bluish card border
  
  // Landing page gradient (bluish, matches header gradient but brighter)
  bgLandingGradient: 'linear-gradient(180deg, #f0f4f8 0%, #e8eef5 25%, #dfe8f2 50%, #e8eef5 75%, #f0f4f8 100%)',
  // Registration page gradient (same style)
  bgRegistrationGradient: 'linear-gradient(180deg, #f0f4f8 0%, #e8eef5 25%, #dfe8f2 50%, #e8eef5 75%, #f0f4f8 100%)',
  
  // Typography
  textTitle: colors.gray[900],  // #0F172A - headlines (dark blue, not pure black)
  textBody: colors.gray[700],   // #334155 - body text
  textMuted: colors.gray[500],  // #64748B - secondary/muted text
  textInverse: '#E6F0FF',       // For dark backgrounds (header)
  
  // Borders & Icons
  border: colors.gray[200],     // #E2E8F0 - subtle borders
  iconDefault: colors.gray[600], // #475569 - default icon color
  iconActive: colors.accent[500], // Cyan for active icons
} as const;

// ==========================================================================
// CSS CUSTOM PROPERTIES
// ==========================================================================
// These are injected into the app and can be used in CSS/styled-components

export const cssVariables = {
  // Accent colors
  '--color-accent-primary': semanticColors.accentPrimary,
  '--color-accent-secondary': semanticColors.accentSecondary,
  
  // Backgrounds
  '--color-bg-main': semanticColors.bgMain,
  '--color-bg-surface': semanticColors.bgSurface,
  '--color-bg-hover': semanticColors.bgHover,
  '--color-bg-card': semanticColors.bgCard,
  '--color-bg-card-border': semanticColors.bgCardBorder,
  
  // Typography
  '--color-text-title': semanticColors.textTitle,
  '--color-text-body': semanticColors.textBody,
  '--color-text-muted': semanticColors.textMuted,
  '--color-text-inverse': semanticColors.textInverse,
  
  // Borders & Icons
  '--color-border': semanticColors.border,
  '--color-icon-default': semanticColors.iconDefault,
  '--color-icon-active': semanticColors.iconActive,
  
  // Semantic
  '--color-success': colors.success[500],
  '--color-error': colors.error[500],
  '--color-warning': colors.warning[500],
} as const;

// ==========================================================================
// TYPE DEFINITIONS
// ==========================================================================

export type ColorShade = 50 | 100 | 200 | 300 | 400 | 500 | 600 | 700 | 800 | 900 | 950;
export type AccentColorName = 'accent' | 'highlight';
export type SemanticColorName = 'success' | 'error' | 'warning';
export type NeutralColorName = 'gray';
export type ColorName = AccentColorName | SemanticColorName | NeutralColorName;

// ==========================================================================
// HELPER FUNCTIONS
// ==========================================================================

export const getAccentColor = (shade: ColorShade = 500) => colors.accent[shade];
export const getHighlightColor = (shade: ColorShade = 500) => colors.highlight[shade];
export const getGrayColor = (shade: ColorShade = 500) => colors.gray[shade];
