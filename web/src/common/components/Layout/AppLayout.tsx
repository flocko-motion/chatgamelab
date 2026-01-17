import { AppShell, Container } from '@mantine/core';
import { AppHeader, type AppHeaderProps, type NavItem } from './AppHeader';
import { AppFooter, type AppFooterProps } from './AppFooter';
import { useResponsiveDesign } from '../../hooks/useResponsiveDesign';

export interface AppLayoutProps {
  children: React.ReactNode;
  variant?: 'public' | 'authenticated';
  navItems?: NavItem[];
  headerProps?: Omit<AppHeaderProps, 'navItems'>;
  footerProps?: AppFooterProps;
  withContainer?: boolean;
  containerSize?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  background?: string;
  transparentFooter?: boolean;
  darkMode?: boolean;
}

const HEADER_HEIGHT = { base: 60, sm: 80 } as const;
const FOOTER_HEIGHT = { base: 50, sm: 60 } as const;

export function AppLayout({
  children,
  variant = 'public',
  navItems,
  headerProps,
  footerProps,
  withContainer = true,
  containerSize = 'xl',
  background = '#ffffff',
  transparentFooter = false,
  darkMode = false,
}: AppLayoutProps) {
  const { isMobile } = useResponsiveDesign();
  const isAuthenticated = variant === 'authenticated';

  return (
    <AppShell
      header={isAuthenticated ? { height: HEADER_HEIGHT } : undefined}
      footer={{ height: FOOTER_HEIGHT }}
      padding={{ base: 'sm', sm: 'md' }}
      style={{
        background,
        minHeight: '100vh',
      }}
    >
      {isAuthenticated && (
        <AppHeader navItems={navItems} {...headerProps} />
      )}

      <AppShell.Main
        pt={isAuthenticated 
          ? (darkMode 
              ? { base: HEADER_HEIGHT.base, sm: HEADER_HEIGHT.sm } 
              : { base: HEADER_HEIGHT.base + 10, sm: HEADER_HEIGHT.sm + 20 })
          : { base: 'sm', sm: 'md' }
        }
        pb={darkMode ? FOOTER_HEIGHT.base : undefined}
        style={darkMode ? { 
          display: 'flex', 
          flexDirection: 'column',
          height: '100vh',
          overflow: 'hidden',
        } : undefined}
      >
        {withContainer ? (
          <Container 
            size={containerSize} 
            px={{ base: 'sm', sm: 'md', lg: 'xl' }}
            py={isMobile ? 'sm' : 'md'}
          >
            {children}
          </Container>
        ) : (
          children
        )}
      </AppShell.Main>

      <AppFooter {...footerProps} transparent={transparentFooter} darkMode={darkMode} />
    </AppShell>
  );
}

export type { NavItem, AppHeaderProps, AppFooterProps };
