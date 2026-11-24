/**
 * Mantine Theme Configuration for ChatGameLab: the "Lab Notebook" look (#567)
 *
 * - Light only: paper page, white cards, ink text and outlines (colors.ts).
 * - Ink is the primary colour: primary buttons, checked controls, focus.
 * - Neon cyan (accent) and magenta (highlight) are accents: fills, shadows,
 *   highlighter strokes. As text use accent.9 / highlight.7 (≥ 4.5:1).
 * - Type: Bricolage Grotesque for headings, DM Sans for text, Space Mono
 *   for small labels; handwriting (Caveat) only as decoration.
 * - Component looks that need hover or focus states live in
 *   theme.module.css and are attached here through classNames.
 *
 * Color usage:
 * - color="ink" (the default) for primary actions; color="accent" for a
 *   cyan fill (ink text on top), color="highlight" sparingly
 * - green/red/orange/blue for semantic states
 * - c="gray.9" titles, c="gray.7" body, c="gray.5" muted
 */

import { createTheme } from '@mantine/core';
import {
  inkColors,
  accentColors,
  highlightColors,
  grayColors,
  violetColors,
  successColors,
  errorColors,
  warningColors,
  infoColors,
  semanticColors,
  layoutColors,
  genreColors,
  genreFallbackColor,
} from './colors';
import classes from './theme.module.css';

// The fonts are served from assets/fonts (fonts.css).
const SANS = "'DM Sans', system-ui, -apple-system, sans-serif";
const DISPLAY = "'Bricolage Grotesque', 'DM Sans', system-ui, sans-serif";
const MONO = "'Space Mono', ui-monospace, SFMono-Regular, Menlo, monospace";

export const mantineTheme = createTheme({
  // ==========================================================================
  // COLOR CONFIGURATION
  // ==========================================================================

  primaryColor: 'ink',
  primaryShade: 5,

  // Readable text on any fill: ink on cyan or magenta, paper on ink.
  autoContrast: true,
  luminanceThreshold: 0.3,

  colors: {
    ink: inkColors,
    accent: accentColors,
    highlight: highlightColors,
    gray: grayColors,
    violet: violetColors,
    green: successColors,
    red: errorColors,
    orange: warningColors,
    blue: infoColors,
  },

  black: inkColors[5],
  white: '#ffffff',

  // ==========================================================================
  // TYPOGRAPHY
  // ==========================================================================

  fontFamily: SANS,
  fontFamilyMonospace: MONO,

  headings: {
    fontFamily: DISPLAY,
    fontWeight: '800',
  },

  // ==========================================================================
  // GENERAL STYLING
  // ==========================================================================

  defaultRadius: 'md',
  focusRing: 'auto',
  cursorType: 'pointer',

  shadows: {
    // Hard offset shadows in ink: the notebook look.
    xs: `2px 2px 0 ${inkColors[5]}`,
    sm: `3px 3px 0 ${inkColors[5]}`,
    md: `4px 4px 0 ${inkColors[5]}`,
    lg: `6px 6px 0 ${inkColors[5]}`,
    xl: `8px 8px 0 ${inkColors[5]}`,
  },

  // ==========================================================================
  // THEME EXTENSIONS (theme.other)
  // ==========================================================================

  other: {
    layout: layoutColors,
    colors: semanticColors,
    genreColors,
    genreFallbackColor,
    fonts: { sans: SANS, display: DISPLAY, mono: MONO, hand: "'Caveat', cursive" },
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
        color: 'ink',
      },
      classNames: { root: classes.button },
    },

    ActionIcon: {
      defaultProps: {
        color: 'ink',
      },
      classNames: { root: classes.actionIcon },
    },

    Anchor: {
      defaultProps: {
        c: 'ink',
      },
      classNames: { root: classes.anchor },
    },

    ThemeIcon: {
      defaultProps: {
        color: 'accent',
      },
    },

    Badge: {
      defaultProps: {
        color: 'ink',
        variant: 'light',
      },
      classNames: { root: classes.badge },
      // Nothing under 12px: the smallest badge keeps readable text.
      vars: (_theme: unknown, props: { size?: string }) =>
        props.size === 'xs' ? { root: { '--badge-fz': '12px', '--badge-height': '20px' } } : { root: {} },
    },

    Loader: {
      defaultProps: {
        color: 'ink',
      },
    },

    Paper: {
      defaultProps: {
        shadow: 'none',
      },
      classNames: { root: classes.paper },
    },

    Card: {
      defaultProps: {
        withBorder: true,
        shadow: 'md',
        radius: 'md',
      },
      classNames: { root: classes.card },
    },

    Modal: {
      classNames: { content: classes.modal, title: classes.modalTitle },
    },

    Menu: {
      classNames: { dropdown: classes.dropdown, item: classes.menuItem },
    },

    Popover: {
      classNames: { dropdown: classes.dropdown },
    },

    Combobox: {
      classNames: { dropdown: classes.dropdown, option: classes.option },
    },

    Alert: {
      classNames: { root: classes.alert, title: classes.alertTitle, message: classes.alertMessage },
    },

    Tooltip: {
      defaultProps: {
        color: 'ink',
      },
    },

    Input: {
      classNames: { input: classes.input },
    },

    SegmentedControl: {
      classNames: { root: classes.segmented, indicator: classes.segmentedIndicator, label: classes.segmentedLabel },
    },

    Tabs: {
      defaultProps: {
        color: 'ink',
      },
    },

    Checkbox: {
      defaultProps: {
        color: 'ink',
      },
    },

    Switch: {
      defaultProps: {
        color: 'accent',
      },
      classNames: { track: classes.switchTrack },
    },

    Table: {
      classNames: { th: classes.th },
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
