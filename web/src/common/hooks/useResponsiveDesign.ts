import { useMantineTheme } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';

/**
 * Mobile breakpoint constant used throughout the app.
 * Below this breakpoint = mobile, at or above = desktop.
 */
export const MOBILE_BREAKPOINT = 'sm' as const;

/**
 * Centralized responsive design hook following Mantine v8 best practices.
 * 
 * Simple binary approach: either mobile or desktop.
 * - Mobile: viewport below 'sm' breakpoint (768px)
 * - Desktop: viewport at or above 'sm' breakpoint
 * 
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { isMobile } = useResponsiveDesign();
 *   
 *   if (isMobile) {
 *     return <MobileView />;
 *   }
 *   return <DesktopView />;
 * }
 * ```
 */
export function useResponsiveDesign() {
  const theme = useMantineTheme();

  // Use em units for better accessibility (respects user font size preferences)
  const isMobile = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);

  return {
    /**
     * True when viewport is below 'sm' breakpoint (768px).
     * Use for mobile-specific layouts like burger menu, stacked cards, etc.
     * If false, we are in desktop mode.
     */
    isMobile: isMobile ?? false,

    /**
     * The breakpoint used for mobile/desktop switch.
     * Use this with hiddenFrom/visibleFrom props for consistency.
     */
    mobileBreakpoint: MOBILE_BREAKPOINT,
  };
}
