import { createTheme } from '@mantine/core';
import { colors } from './colors';

export const mantineTheme = createTheme({
  primaryColor: 'violet',
  fontFamily: 'Inter, system-ui, -apple-system, sans-serif',

  other: {
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
  },
  
  // Override color palettes with our centralized colors
  colors: {
    violet: [
      colors.primary[50],   // 50
      colors.primary[100],  // 100
      colors.primary[200],  // 200
      colors.primary[300],  // 300
      colors.primary[400],  // 400
      colors.primary[500],  // 500 - main primary color
      colors.primary[600],  // 600 - hover
      colors.primary[700],  // 700 - dark
      colors.primary[800],  // 800
      colors.primary[900],  // 900
      colors.primary[950],  // 950
    ],
    
    green: [
      colors.success[50],   // 50
      colors.success[100],  // 100
      colors.success[200],  // 200
      colors.success[300],  // 300
      colors.success[400],  // 400
      colors.success[500],  // 500 - main success color
      colors.success[600],  // 600
      colors.success[700],  // 700
      colors.success[800],  // 800
      colors.success[900],  // 900
      colors.success[950],  // 950
    ],
    
    red: [
      colors.error[50],    // 50
      colors.error[100],   // 100
      colors.error[200],   // 200
      colors.error[300],   // 300
      colors.error[400],   // 400
      colors.error[500],   // 500 - main error color
      colors.error[600],   // 600
      colors.error[700],   // 700
      colors.error[800],   // 800
      colors.error[900],   // 900
      colors.error[950],   // 950
    ],
    
    orange: [
      colors.warning[50],   // 50
      colors.warning[100],  // 100
      colors.warning[200],  // 200
      colors.warning[300],  // 300
      colors.warning[400],  // 400
      colors.warning[500],  // 500 - main warning color
      colors.warning[600],  // 600
      colors.warning[700],  // 700
      colors.warning[800],  // 800
      colors.warning[900],  // 900
      colors.warning[950],  // 950
    ],
    
    gray: [
      colors.gray[50],     // 50
      colors.gray[100],    // 100
      colors.gray[200],    // 200
      colors.gray[300],    // 300
      colors.gray[400],    // 400
      colors.gray[500],    // 500
      colors.gray[600],    // 600
      colors.gray[700],    // 700
      colors.gray[800],    // 800
      colors.gray[900],    // 900
      colors.gray[950],    // 950
    ],
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
        color: 'violet',
      },
      styles: {
        root: {
          transition: 'all 0.2s ease',
        },
      },
    },
    
    ActionIcon: {
      defaultProps: {
        color: 'violet',
      },
    },
    
    Anchor: {
      defaultProps: {
        c: 'violet',
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
            borderColor: 'var(--mantine-color-violet-6)',
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
