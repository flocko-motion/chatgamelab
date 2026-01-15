import { createRootRoute, Outlet, useNavigate } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { Center, Loader, useMantineTheme } from '@mantine/core';
import { IconPlayerPlay, IconEdit, IconBuilding, IconUsers } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useEffect } from 'react';
import { AppLayout, type NavItem } from '../common/components/Layout';
import { useAuth } from '../providers/AuthProvider';
import { RegistrationForm } from '../features/auth';
import { useLocation } from '@tanstack/react-router';
import { ROUTES } from '../common/routes/routes';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const { isLoading, needsRegistration, registrationData, isAuthenticated, backendUser } = useAuth();
  const { t } = useTranslation('navigation');
  const location = useLocation();
  const navigate = useNavigate();
  const theme = useMantineTheme();
  const pathname = location.pathname;
  const isHomePage = pathname === ROUTES.HOME;
  
  // Public routes that don't require authentication
  const isPublicRoute = isHomePage || pathname.startsWith(ROUTES.AUTH_LOGIN);

  // Determine layout variant based on auth state
  const isFullyAuthenticated = isAuthenticated && backendUser && !needsRegistration;
  const useAuthenticatedLayout = isFullyAuthenticated && !isHomePage;
  
  // Redirect to homepage if trying to access protected route without full authentication
  const shouldRedirect = !isLoading && !isFullyAuthenticated && !isPublicRoute && !needsRegistration;
  
  // All hooks must be called before any early returns
  useEffect(() => {
    if (shouldRedirect) {
      navigate({ to: ROUTES.HOME });
    }
  }, [shouldRedirect, navigate]);

  // Navigation items for authenticated header
  const navItems: NavItem[] = [
    { label: t('dashboard'), icon: <IconBuilding size={18} />, onClick: () => navigate({ to: ROUTES.DASHBOARD }) },
    { label: t('play'), icon: <IconPlayerPlay size={18} />, onClick: () => navigate({ to: ROUTES.SESSIONS as '/' }) },
    { label: t('create'), icon: <IconEdit size={18} />, onClick: () => navigate({ to: ROUTES.CREATIONS as '/' }) },
  ];

  // Header navigation callbacks
  const headerProps = useAuthenticatedLayout ? {
    onSettingsClick: () => navigate({ to: ROUTES.SETTINGS }),
    onProfileClick: () => navigate({ to: ROUTES.PROFILE }),
    onApiKeysClick: () => navigate({ to: ROUTES.API_KEYS }),
  } : undefined;

  // Show loading state while auth is initializing
  if (isLoading) {
    return (
      <AppLayout variant="public">
        <Center h="50vh">
          <Loader size="lg" />
        </Center>
      </AppLayout>
    );
  }

  // Show registration form if user needs to complete registration
  if (needsRegistration && registrationData) {
    return (
      <>
        <AppLayout variant="public" background={theme.other.colors.bgRegistrationGradient}>
          <RegistrationForm registrationData={registrationData} />
        </AppLayout>
        <TanStackRouterDevtools position="bottom-right" />
      </>
    );
  }

  // Show loading while redirecting
  if (shouldRedirect) {
    return (
      <AppLayout variant="public">
        <Center h="50vh">
          <Loader size="lg" />
        </Center>
      </AppLayout>
    );
  }

  return (
    <>
      <AppLayout 
        variant={useAuthenticatedLayout ? 'authenticated' : 'public'}
        navItems={useAuthenticatedLayout ? navItems : undefined}
        background={theme.other.colors.bgLandingGradient}
        transparentFooter={isHomePage}
        headerProps={headerProps}
      >
        <Outlet />
      </AppLayout>
      <TanStackRouterDevtools position="bottom-right" />
    </>
  );
}
