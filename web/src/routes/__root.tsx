import { createRootRoute, Outlet, useNavigate } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { Center, Loader, useMantineTheme } from '@mantine/core';
import { IconPlayerPlay, IconWorld, IconHome, IconBuilding, IconUsers } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useEffect } from 'react';
import { AppLayout, type NavItem } from '../common/components/Layout';
import { useAuth } from '../providers/AuthProvider';
import { RegistrationForm } from '../features/auth';
import { useLocation } from '@tanstack/react-router';
import { ROUTES } from '../common/routes/routes';
import { isAdmin } from '../common/lib/roles';

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
  
  // Game player routes need special dark styling
  const isGamePlayerRoute = pathname.includes('/play') || pathname.startsWith('/sessions/');
  
  // Public routes that don't require authentication
  const isPublicRoute = isHomePage || pathname.startsWith(ROUTES.AUTH_LOGIN);

  // Determine layout variant based on auth state
  const isFullyAuthenticated = isAuthenticated && backendUser && !needsRegistration;
  const useAuthenticatedLayout = isFullyAuthenticated && !isHomePage;
  
  // Redirect to homepage only if NOT authenticated (not just waiting for backend user)
  // If authenticated but backendUser is still loading, keep showing loader instead of redirecting
  const shouldRedirect = !isLoading && !isAuthenticated && !isPublicRoute && !needsRegistration;
  
  // All hooks must be called before any early returns
  useEffect(() => {
    if (shouldRedirect) {
      navigate({ to: ROUTES.HOME });
    }
  }, [shouldRedirect, navigate]);

  // Navigation items for authenticated header
  const navItems: NavItem[] = [
    { 
      label: t('dashboard'), 
      icon: <IconHome size={18} />, 
      onClick: () => navigate({ to: ROUTES.DASHBOARD }),
      active: pathname === ROUTES.DASHBOARD,
    },
    { 
      label: t('myGames'), 
      icon: <IconPlayerPlay size={18} />, 
      onClick: () => navigate({ to: ROUTES.MY_GAMES as '/' }),
      active: pathname.startsWith(ROUTES.MY_GAMES),
    },
    { 
      label: t('allGames'), 
      icon: <IconWorld size={18} />, 
      onClick: () => navigate({ to: ROUTES.ALL_GAMES as '/' }),
      active: pathname.startsWith(ROUTES.ALL_GAMES) || pathname.startsWith(ROUTES.SESSIONS),
    },
  ];

  // Admin-only navigation items
  if (isAdmin(backendUser)) {
    navItems.push(
      { 
        label: t('manageOrganizations'), 
        icon: <IconBuilding size={18} />, 
        onClick: () => navigate({ to: ROUTES.ADMIN_ORGANIZATIONS as '/' }),
        active: pathname.startsWith(ROUTES.ADMIN_ORGANIZATIONS),
      },
      { 
        label: t('manageUsers'), 
        icon: <IconUsers size={18} />, 
        onClick: () => navigate({ to: ROUTES.ADMIN_USERS as '/' }),
        active: pathname.startsWith(ROUTES.ADMIN_USERS),
      },
    );
  }

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

  // Game player uses dimmed background
  const layoutBackground = isGamePlayerRoute 
    ? '#e8e8ec' 
    : theme.other.colors.bgLandingGradient;

  return (
    <>
      <AppLayout 
        variant={useAuthenticatedLayout ? 'authenticated' : 'public'}
        navItems={useAuthenticatedLayout ? navItems : undefined}
        background={layoutBackground}
        transparentFooter={isHomePage}
        headerProps={headerProps}
        darkMode={isGamePlayerRoute}
        withContainer={!isGamePlayerRoute}
      >
        <Outlet />
      </AppLayout>
      <TanStackRouterDevtools position="bottom-right" />
    </>
  );
}
