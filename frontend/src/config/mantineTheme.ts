/**
 * Mantine Theme Configuration for ChatGameLab
 * 
 * This theme enforces a light color scheme with:
 * - Cyan (accent) as primary color
 * - Auto-contrast for readable button text
 * - Consistent component styling
 * 
 * Color Usage:
 * - Use color="accent" for primary actions
 * - Use color="highlight" sparingly for attention
 * - Use color="green/red/orange/blue" for semantic states
 * - Use c="gray.9" for title text, c="gray.7" for body, c="gray.5" for muted
 */

import { createTheme } from '@mantine/core';
import {
  accentColors,
  highlightColors,
  grayColors,
  successColors,
  errorColors,
  warningColors,
  infoColors,
  semanticColors,
  layoutColors,
} from './colors';

export const mantineTheme = createTheme({
  // ==========================================================================
  // COLOR CONFIGURATION
  // ==========================================================================
  
  primaryColor: 'accent',
  primaryShade: 5,
  
  // Auto-contrast ensures readable text on colored backgrounds (e.g., cyan buttons)
  autoContrast: true,
  luminanceThreshold: 0.3,
  
  // Custom color palettes
  colors: {
    accent: accentColors,
    highlight: highlightColors,
    gray: grayColors,
    green: successColors,
    red: errorColors,
    orange: warningColors,
    blue: infoColors,
  },

  // ==========================================================================
  // TYPOGRAPHY
  // ==========================================================================
  
  fontFamily: 'Inter, system-ui, -apple-system, sans-serif',
  
  headings: {
    fontFamily: 'Inter, system-ui, -apple-system, sans-serif',
    fontWeight: '600',
  },

  // ==========================================================================
  // GENERAL STYLING
  // ==========================================================================
  
  defaultRadius: 'md',
  focusRing: 'auto',
  cursorType: 'pointer',

  // ==========================================================================
  // THEME EXTENSIONS (theme.other)
  // ==========================================================================
  
  other: {
    // Layout colors for header/dark elements
    layout: layoutColors,
    // Semantic colors for direct access
    colors: semanticColors,
  },

  // ==========================================================================
  // COMPONENT DEFAULTS
  // ==========================================================================
  
  components: {
    Container: {
      defaultProps: {
        size: 'xl',
      },
      vars: (_theme: unknown, props: { size?: string }) => ({
        root: {
          // Use max-width in px - on small screens, container naturally uses 100% minus padding
          // xl increased by 1/3 (1400 → 1867), others scaled proportionally
          '--container-size': props.size === 'xs' ? '720px'
            : props.size === 'sm' ? '960px'
            : props.size === 'md' ? '1280px'
            : props.size === 'lg' ? '1520px'
            : props.size === 'xl' ? '1867px'
            : '1520px',
        },
      }),
    },
    
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
        c: 'accent.6',
      },
      styles: {
        root: {
          textDecoration: 'none',
          transition: 'color 0.15s ease',
          '&:hover': {
            textDecoration: 'underline',
          },
        },
      },
    },
    
    ThemeIcon: {
      defaultProps: {
        color: 'accent',
      },
    },
    
    Badge: {
      defaultProps: {
        color: 'accent',
      },
    },
    
    Loader: {
      defaultProps: {
        color: 'accent',
      },
    },
    
    Paper: {
      defaultProps: {
        shadow: 'sm',
      },
    },
    
    Card: {
      defaultProps: {
        withBorder: true,
      },
      styles: {
        root: {
          borderColor: 'var(--mantine-color-gray-2)',
        },
      },
    },
    
    // Prevent iOS Safari auto-zoom on input focus (triggers when font-size < 16px)
    // Setting size="md" ensures 16px font-size
    TextInput: {
      defaultProps: {
        size: 'md',
      },
    },
    
    PasswordInput: {
      defaultProps: {
        size: 'md',
      },
    },
    
    Textarea: {
      defaultProps: {
        size: 'md',
      },
    },
    
    Select: {
      defaultProps: {
        size: 'md',
      },
    },
    
    MultiSelect: {
      defaultProps: {
        size: 'md',
      },
    },
    
    NumberInput: {
      defaultProps: {
        size: 'md',
      },
    },
    
    Autocomplete: {
      defaultProps: {
        size: 'md',
      },
    },
    
  },
});
