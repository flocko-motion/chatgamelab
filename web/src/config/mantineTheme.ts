import { createTheme } from '@mantine/core';
import type { MantineColorsTuple } from '@mantine/core';
import { colors, semanticColors } from './colors';

// Convert our color scales to Mantine's tuple format (10 shades, index 0-9)
// Mantine uses index 5 as the "main" color by default
const accentTuple: MantineColorsTuple = [
  colors.accent[50],
  colors.accent[100],
  colors.accent[200],
  colors.accent[300],
  colors.accent[400],
  colors.accent[500],  // Main accent - cyan
  colors.accent[600],
  colors.accent[700],
  colors.accent[800],
  colors.accent[900],
];

const highlightTuple: MantineColorsTuple = [
  colors.highlight[50],
  colors.highlight[100],
  colors.highlight[200],
  colors.highlight[300],
  colors.highlight[400],
  colors.highlight[500],  // Main highlight - magenta
  colors.highlight[600],
  colors.highlight[700],
  colors.highlight[800],
  colors.highlight[900],
];

const grayTuple: MantineColorsTuple = [
  colors.gray[50],
  colors.gray[100],
  colors.gray[200],
  colors.gray[300],
  colors.gray[400],
  colors.gray[500],
  colors.gray[600],
  colors.gray[700],
  colors.gray[800],
  colors.gray[900],
];

const successTuple: MantineColorsTuple = [
  colors.success[50],
  colors.success[100],
  colors.success[200],
  colors.success[300],
  colors.success[400],
  colors.success[500],
  colors.success[600],
  colors.success[700],
  colors.success[800],
  colors.success[900],
];

const errorTuple: MantineColorsTuple = [
  colors.error[50],
  colors.error[100],
  colors.error[200],
  colors.error[300],
  colors.error[400],
  colors.error[500],
  colors.error[600],
  colors.error[700],
  colors.error[800],
  colors.error[900],
];

const warningTuple: MantineColorsTuple = [
  colors.warning[50],
  colors.warning[100],
  colors.warning[200],
  colors.warning[300],
  colors.warning[400],
  colors.warning[500],
  colors.warning[600],
  colors.warning[700],
  colors.warning[800],
  colors.warning[900],
];

export const mantineTheme = createTheme({
  // Use 'accent' as primary color (cyan from logo)
  primaryColor: 'accent',
  primaryShade: 5, // Use index 5 (accent[500]) for filled buttons
  fontFamily: 'Inter, system-ui, -apple-system, sans-serif',

  other: {
    // Header/dark theme elements (keep gradient header as-is)
    layout: {
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
    },
    // Semantic color roles for easy access in components
    colors: semanticColors,
  },
  
  // Custom color palettes
  colors: {
    // Primary accent - cyan (use for primary actions, focus states)
    accent: accentTuple,
    
    // Secondary accent - magenta (use for notifications, badges, special actions)
    highlight: highlightTuple,
    
    // Neutrals
    gray: grayTuple,
    
    // Semantic colors
    green: successTuple,
    red: errorTuple,
    orange: warningTuple,
  },
  
  // Default styling
  defaultRadius: 'md',
  
  // Focus styles
  focusRing: 'always',
  scale: 1,
  
  // Cursor styles
  cursorType: 'pointer',
  
  // Component-specific overrides
  components: {
    Button: {
      defaultProps: {
        color: 'accent',
      },
      styles: {
        root: {
          transition: 'all 0.2s ease',
        },
      },
    },
    
    ActionIcon: {
      defaultProps: {
        color: 'accent',
      },
    },
    
    Anchor: {
      defaultProps: {
        c: 'accent',
      },
      styles: {
        root: {
          textDecoration: 'none',
          transition: 'all 0.2s ease',
          '&:hover': {
            textDecoration: 'underline',
          },
        },
      },
    },
    
    TextInput: {
      styles: {
        input: {
          '&:focus': {
            borderColor: 'var(--mantine-color-accent-6)',
          },
        },
      },
    },
    
    Paper: {
      defaultProps: {
        shadow: 'sm',
      },
    },
    
    Card: {
      styles: {
        root: {
          transition: 'all 0.2s ease',
          '&:hover': {
            transform: 'translateY(-1px)',
            boxShadow: 'var(--mantine-shadow-lg)',
          },
        },
      },
    },
  },
});
