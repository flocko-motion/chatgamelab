// Color palette configuration for ChatGameLab
// Centralized color definitions make it easy to change the app's visual theme

export const colors = {
  // Primary accent color - change this to update the entire app's color scheme
  primary: {
    50: '#f5f3ff',
    100: '#e9e5ff',
    200: '#d4d0ff',
    300: '#b8b3ff',
    400: '#9c95ff',
    500: '#8077ff', // Main primary color
    600: '#6d62ff',
    700: '#5a4dff',
    800: '#4738ff',
    900: '#3425ff',
    950: '#1f1aff',
  },

  // Semantic colors
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

  // Neutral colors
  gray: {
    50: '#f9fafb',
    100: '#f3f4f6',
    200: '#e5e7eb',
    300: '#d1d5db',
    400: '#9ca3af',
    500: '#6b7280',
    600: '#4b5563',
    700: '#374151',
    800: '#1f2937',
    900: '#111827',
    950: '#030712',
  },

  // Special colors
  white: '#ffffff',
  black: '#000000',
  transparent: 'transparent',
} as const;

// Mantine color mappings
export const mantineColors = {
  primary: colors.primary[500],
  primaryHover: colors.primary[600],
  primaryLight: colors.primary[100],
  primaryDark: colors.primary[700],
  
  success: colors.success[500],
  error: colors.error[500],
  warning: colors.warning[500],
  
  gray: colors.gray[500],
  grayLight: colors.gray[400],
  grayDark: colors.gray[600],
  
  background: colors.white,
  surface: colors.gray[50],
  border: colors.gray[200],
  text: colors.gray[900],
  textSecondary: colors.gray[600],
  textMuted: colors.gray[400],
} as const;

// CSS custom properties for global access
export const cssVariables = {
  '--color-primary': mantineColors.primary,
  '--color-primary-hover': mantineColors.primaryHover,
  '--color-primary-light': mantineColors.primaryLight,
  '--color-primary-dark': mantineColors.primaryDark,
  
  '--color-success': mantineColors.success,
  '--color-error': mantineColors.error,
  '--color-warning': mantineColors.warning,
  
  '--color-gray': mantineColors.gray,
  '--color-gray-light': mantineColors.grayLight,
  '--color-gray-dark': mantineColors.grayDark,
  
  '--color-background': mantineColors.background,
  '--color-surface': mantineColors.surface,
  '--color-border': mantineColors.border,
  '--color-text': mantineColors.text,
  '--color-text-secondary': mantineColors.textSecondary,
  '--color-text-muted': mantineColors.textMuted,
} as const;

// Type definitions
export type ColorShade = keyof typeof colors.primary;
export type ColorName = keyof typeof colors;

// Helper functions
export const getColor = (colorName: ColorName, shade: ColorShade = 500) => {
  return colors[colorName]?.[shade] || colors.gray[shade];
};

export const getPrimaryColor = (shade: ColorShade = 500) => {
  return colors.primary[shade];
};

// Theme configuration for Mantine
export const themeColors = {
  primary: mantineColors.primary,
  // You can easily change the primary color here:
  // primary: colors.blue[500], // for blue theme
  // primary: colors.green[500], // for green theme
  // primary: colors.red[500], // for red theme
};
