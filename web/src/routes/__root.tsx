import { createRootRoute, Outlet, useNavigate } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { Center, Loader, useMantineTheme } from "@mantine/core";
import {
  IconPlayerPlay,
  IconWorld,
  IconHome,
  IconBuilding,
  IconUsers,
  IconKey,
  IconSchool,
  IconSettings,
  IconTools,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useEffect } from "react";
import { AppLayout, type NavItem } from "../common/components/Layout";
import { InactiveWorkshopMessage } from "../common/components/InactiveWorkshopMessage";
import { useAuth } from "../providers/AuthProvider";
import { useWorkshopMode } from "../providers/WorkshopModeProvider";
import { RegistrationForm } from "../features/auth";
import { useLocation } from "@tanstack/react-router";
import { ROUTES } from "../common/routes/routes";
import {
  isAdmin,
  getUserInstitutionId,
  hasRole,
  Role,
} from "../common/lib/roles";

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const {
    isLoading,
    needsRegistration,
    registrationData,
    isAuthenticated,
    backendUser,
    isParticipant,
    isWorkshopInactive,
    logout,
  } = useAuth();
  const { isInWorkshopMode, activeWorkshopName, exitWorkshopMode } =
    useWorkshopMode();
  const { t } = useTranslation("navigation");
  const location = useLocation();
  const navigate = useNavigate();
  const theme = useMantineTheme();
  const pathname = location.pathname;
  const isHomePage = pathname === ROUTES.HOME;

  // Workshop mode: true participants OR staff/head who entered workshop mode
  const isInWorkshopUI = isParticipant || isInWorkshopMode;

  // Game player routes need special dark styling
  const isGamePlayerRoute =
    pathname.includes("/play") || pathname.startsWith("/sessions/");

  // Public routes that don't require authentication
  const isPublicRoute =
    isHomePage ||
    pathname.startsWith(ROUTES.AUTH_LOGIN) ||
    pathname.startsWith(ROUTES.INVITES);

  // Routes that participants are allowed to access
  const isParticipantAllowedRoute =
    pathname === ROUTES.MY_WORKSHOP ||
    pathname.startsWith("/sessions/") ||
    pathname.includes("/play");

  // Determine layout variant based on auth state
  const isFullyAuthenticated =
    isAuthenticated && backendUser && !needsRegistration;
  const useAuthenticatedLayout = isFullyAuthenticated && !isHomePage;

  // Redirect to homepage only if NOT authenticated (not just waiting for backend user)
  // If authenticated but backendUser is still loading, keep showing loader instead of redirecting
  const shouldRedirect =
    !isLoading && !isAuthenticated && !isPublicRoute && !needsRegistration;

  // Redirect participants to my-workshop if they try to access other routes
  const shouldRedirectParticipant =
    !isLoading && isParticipant && !isPublicRoute && !isParticipantAllowedRoute;

  // Redirect workshop mode users from dashboard to my-workshop
  const shouldRedirectWorkshopMode =
    !isLoading &&
    isInWorkshopMode &&
    pathname === ROUTES.DASHBOARD;

  // All hooks must be called before any early returns
  useEffect(() => {
    if (shouldRedirect) {
      navigate({ to: ROUTES.HOME });
    }
  }, [shouldRedirect, navigate]);

  // Redirect participants to my-workshop
  useEffect(() => {
    if (shouldRedirectParticipant) {
      navigate({ to: ROUTES.MY_WORKSHOP as "/" });
    }
  }, [shouldRedirectParticipant, navigate]);

  // Redirect workshop mode users from dashboard to my-workshop
  useEffect(() => {
    if (shouldRedirectWorkshopMode) {
      navigate({ to: ROUTES.MY_WORKSHOP as "/" });
    }
  }, [shouldRedirectWorkshopMode, navigate]);

  // Organization/Workshop permissions - needed early for nav items
  const userInstitutionId = getUserInstitutionId(backendUser);
  const canManageOrgApiKeys =
    hasRole(backendUser, Role.Head) || hasRole(backendUser, Role.Staff);

  // Navigation items for authenticated header
  // Workshop mode users (participants OR staff/head in workshop mode) see "My Workshop"
  // Staff/Head in workshop mode also see "Workshop Settings" to manage workshops
  const workshopNavItems: NavItem[] = [
    {
      label: t("myWorkshop"),
      icon: <IconSchool size={18} />,
      onClick: () => navigate({ to: ROUTES.MY_WORKSHOP as "/" }),
      active:
        pathname === ROUTES.MY_WORKSHOP ||
        pathname.startsWith("/sessions/") ||
        pathname.includes("/play"),
    },
  ];

  // Add Create Game for Staff/Head in workshop mode (not for participants)
  if (!isParticipant) {
    workshopNavItems.push({
      label: t("createGame"),
      icon: <IconTools size={18} />,
      onClick: () => navigate({ to: ROUTES.MY_GAME_CREATE as "/" }),
      active: false,
    });
  }

  // Add Workshop Settings for Staff/Head in workshop mode (not for participants)
  if (isInWorkshopMode && !isParticipant && canManageOrgApiKeys) {
    workshopNavItems.push({
      label: t("workshopSettings"),
      icon: <IconSettings size={18} />,
      onClick: () => navigate({ to: ROUTES.MY_WORKSHOP_SETTINGS as "/" }),
      active: pathname === ROUTES.MY_WORKSHOP_SETTINGS,
    });
  }

  const navItems: NavItem[] = isInWorkshopUI
    ? workshopNavItems
    : [
        {
          label: t("dashboard"),
          icon: <IconHome size={18} />,
          onClick: () => navigate({ to: ROUTES.DASHBOARD }),
          active: pathname === ROUTES.DASHBOARD,
        },
        {
          label: t("createGame"),
          icon: <IconTools size={18} />,
          onClick: () => navigate({ to: ROUTES.MY_GAME_CREATE as "/" }),
          active: false,
        },
        {
          label: t("myGames"),
          icon: <IconPlayerPlay size={18} />,
          onClick: () => navigate({ to: ROUTES.MY_GAMES as "/" }),
          active:
            pathname.startsWith(ROUTES.MY_GAMES) &&
            pathname !== ROUTES.MY_GAME_CREATE,
        },
        {
          label: t("allGames"),
          icon: <IconWorld size={18} />,
          onClick: () => navigate({ to: ROUTES.ALL_GAMES as "/" }),
          active:
            pathname.startsWith(ROUTES.ALL_GAMES) ||
            pathname.startsWith(ROUTES.SESSIONS),
        },
      ];

  // Organization navigation - visible only if user has an organization and is NOT in workshop mode
  // Workshop mode users should not see organization details
  if (userInstitutionId && !isInWorkshopUI) {
    // Build organization sub-items
    const orgChildren: NavItem[] = [
      {
        label: t("orgMembers"),
        icon: <IconUsers size={18} />,
        onClick: () => navigate({ to: ROUTES.MY_ORGANIZATION as "/" }),
        active: pathname === ROUTES.MY_ORGANIZATION,
      },
    ];

    // Add Workshops for heads and staff
    if (canManageOrgApiKeys) {
      orgChildren.push({
        label: t("orgWorkshops"),
        icon: <IconSchool size={18} />,
        onClick: () =>
          navigate({ to: ROUTES.MY_ORGANIZATION_WORKSHOPS as "/" }),
        active: pathname === ROUTES.MY_ORGANIZATION_WORKSHOPS,
      });
    }

    // Add API Keys only for heads and staff
    if (canManageOrgApiKeys) {
      orgChildren.push({
        label: t("orgApiKeys"),
        icon: <IconKey size={18} />,
        onClick: () => navigate({ to: ROUTES.MY_ORGANIZATION_API_KEYS as "/" }),
        active: pathname === ROUTES.MY_ORGANIZATION_API_KEYS,
      });
    }

    navItems.push({
      label: t("myOrganization"),
      icon: <IconBuilding size={18} />,
      active: pathname.startsWith(ROUTES.MY_ORGANIZATION),
      children: orgChildren,
    });
  }

  // Admin-only navigation items
  if (isAdmin(backendUser)) {
    navItems.push(
      {
        label: t("manageOrganizations"),
        icon: <IconBuilding size={18} />,
        onClick: () => navigate({ to: ROUTES.ADMIN_ORGANIZATIONS as "/" }),
        active: pathname.startsWith(ROUTES.ADMIN_ORGANIZATIONS),
      },
      {
        label: t("manageUsers"),
        icon: <IconUsers size={18} />,
        onClick: () => navigate({ to: ROUTES.ADMIN_USERS as "/" }),
        active: pathname.startsWith(ROUTES.ADMIN_USERS),
      },
    );
  }

  // Header navigation callbacks
  // Participants have minimal access, staff/head/individual in workshop mode keep full user actions
  const headerProps = useAuthenticatedLayout
    ? isInWorkshopUI
      ? {
          isParticipant: isParticipant, // True participants show minimal header
          isInWorkshopMode: isInWorkshopMode, // Staff/head/individual in workshop mode
          workshopName: activeWorkshopName,
          onExitWorkshopMode: isInWorkshopMode
            ? async () => {
                await exitWorkshopMode();
                navigate({ to: ROUTES.MY_ORGANIZATION_WORKSHOPS as "/" });
              }
            : undefined,
          // Staff/head/individual in workshop mode keep access to profile/settings/api-keys
          ...(isInWorkshopMode && {
            onSettingsClick: () => navigate({ to: ROUTES.SETTINGS }),
            onProfileClick: () => navigate({ to: ROUTES.PROFILE }),
            onApiKeysClick: () => navigate({ to: ROUTES.API_KEYS }),
          }),
        }
      : {
          onSettingsClick: () => navigate({ to: ROUTES.SETTINGS }),
          onProfileClick: () => navigate({ to: ROUTES.PROFILE }),
          onApiKeysClick: () => navigate({ to: ROUTES.API_KEYS }),
        }
    : undefined;

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

  // Show inactive workshop message for participants
  if (isWorkshopInactive) {
    return (
      <AppLayout variant="public">
        <InactiveWorkshopMessage onLogout={logout} />
      </AppLayout>
    );
  }

  // Show registration form if user needs to complete registration
  if (needsRegistration && registrationData) {
    return (
      <>
        <AppLayout
          variant="public"
          background={theme.other.colors.bgRegistrationGradient}
        >
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
    ? "#e8e8ec"
    : theme.other.colors.bgLandingGradient;

  return (
    <>
      <AppLayout
        variant={useAuthenticatedLayout ? "authenticated" : "public"}
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
