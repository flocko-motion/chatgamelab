import {
  AppShell,
  Group,
  Box,
  Burger,
  Drawer,
  Image,
  UnstyledButton,
  useMantineTheme,
} from "@mantine/core";
import { useDisclosure, useElementSize } from "@mantine/hooks";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { useBranding } from "@/providers/branding";
import { useResponsiveDesign } from "../../../hooks/useResponsiveDesign";
import { ROUTES } from "../../../routes/routes";
import logoLandscape from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Landscape-Colorful-Black-Text-Transparent.png";

import type { AppHeaderProps } from "./types";
import { DesktopNavigation } from "./DesktopNavigation";
import { UserActions } from "./UserActions";
import { NotificationBell } from "../../NotificationBell";
import { MobileTabBar } from "../MobileTabBar";
import {
  GuestMobileNavigation,
  WorkshopModeMobileNavigation,
  MobileNavigation,
} from "./MobileNavigation";

export function AppHeader({
  navItems = [],
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
  isGuest = false,
  isSignedOut = false,
  isInWorkshopMode = false,
  workshopName,
  onExitWorkshopMode,
}: AppHeaderProps) {
  const [mobileNavOpened, { open: openMobileNav, close: closeMobileNav }] =
    useDisclosure(false);
  const { isMobile: isViewportMobile } = useResponsiveDesign();
  const { t } = useTranslation("common");
  const branding = useBranding();
  const navigate = useNavigate();
  const theme = useMantineTheme();

  // Measure header container width
  const { ref: headerRef, width: headerWidth } = useElementSize();
  const { ref: measureNavRef, width: navWidth } = useElementSize();
  const { ref: measureLogoRef, width: logoWidth } = useElementSize();
  const { ref: measureActionsRef, width: actionsWidth } = useElementSize();

  // With left-aligned layout, we only need to check if total content fits
  const contentOverflows = (() => {
    if (
      headerWidth === 0 ||
      navWidth === 0 ||
      logoWidth === 0 ||
      actionsWidth === 0
    )
      return false;

    const padding = 32; // p="md" = 16px * 2
    const gap = 16; // gap="md" between left and right sections
    const totalWidth = logoWidth + navWidth + actionsWidth + gap + padding;

    return totalWidth > headerWidth;
  })();

  // Force mobile if viewport is mobile OR content overflows
  const forceMobile = isViewportMobile || contentOverflows;

  useEffect(() => {
    if (!forceMobile && mobileNavOpened) {
      closeMobileNav();
    }
  }, [forceMobile, mobileNavOpened, closeMobileNav]);

  return (
    <>
      {/* Hidden measurement container - always rendered to measure content */}
      <Box
        style={{
          position: "absolute",
          visibility: "hidden",
          pointerEvents: "none",
          height: 0,
          overflow: "hidden",
        }}
        aria-hidden="true"
      >
        <Box ref={measureNavRef} style={{ display: "inline-block" }}>
          <DesktopNavigation items={navItems} />
        </Box>
        <Box ref={measureLogoRef} style={{ display: "inline-block" }}>
          <Image
            src={logoLandscape}
            alt=""
            h={50}
            w="auto"
            fit="contain"
          />
        </Box>
        <Box ref={measureActionsRef} style={{ display: "inline-block" }}>
          <UserActions
            onSettingsClick={onSettingsClick}
            onProfileClick={onProfileClick}
            onApiKeysClick={onApiKeysClick}
            onLogoutClick={onLogoutClick}
            isGuest={isGuest}
            isSignedOut={isSignedOut}
            isInWorkshopMode={isInWorkshopMode}
            workshopName={workshopName}
            onExitWorkshopMode={onExitWorkshopMode}
          />
        </Box>
      </Box>

      <AppShell.Header
        ref={headerRef}
        p="md"
        style={{
          background: theme.other.layout.headerGradient,
          borderBottom: theme.other.layout.borderLight,
          boxShadow: theme.other.layout.shadowHeader,
        }}
      >
        <Group
          justify="space-between"
          align="center"
          h="100%"
          wrap="nowrap"
          gap="md"
        >
          {/* Left section: Logo + Navigation (desktop) */}
          <Group gap="lg" align="center" wrap="nowrap">
            {/* Logo - navigates to workshop for Guests/workshop mode, dashboard otherwise */}
            <UnstyledButton
              onClick={() =>
                navigate({
                  to: (isGuest || isInWorkshopMode
                    ? ROUTES.MY_WORKSHOP
                    : ROUTES.DASHBOARD) as "/",
                })
              }
              aria-label={t("header.goToDashboard")}
              style={{
                borderRadius: "var(--mantine-radius-sm)",
                transition: "background-color 150ms ease",
              }}
              styles={{
                root: {
                  "&:hover": {
                    backgroundColor: theme.other.layout.bgHover,
                  },
                  "&:active": {
                    backgroundColor: theme.other.layout.bgActive,
                  },
                  "&:focusVisible": {
                    outline: `2px solid ${theme.colors.accent[6]}`,
                    outlineOffset: 2,
                  },
                },
              }}
            >
              <Image
                src={logoLandscape}
                alt={branding.siteName}
                h={{ base: 36, sm: 50 }}
                w="auto"
                fit="contain"
              />
            </UnstyledButton>

            {/* Desktop: Nav items */}
            {!forceMobile && <DesktopNavigation items={navItems} />}
          </Group>

          {/* Right section */}
          <Group gap="sm" align="center" wrap="nowrap">
            {/* Desktop: Full user actions, or always inline when signed out */}
            {(!forceMobile || isSignedOut) && (
              <UserActions
                onSettingsClick={onSettingsClick}
                onProfileClick={onProfileClick}
                onApiKeysClick={onApiKeysClick}
                onLogoutClick={onLogoutClick}
                isGuest={isGuest}
                isSignedOut={isSignedOut}
                isInWorkshopMode={isInWorkshopMode}
                workshopName={workshopName}
                onExitWorkshopMode={onExitWorkshopMode}
              />
            )}

            {/* Mobile: the bell (not for guests), then the burger menu (not
                when signed out) */}
            {forceMobile && !isSignedOut && !isGuest && <NotificationBell />}
            {forceMobile && !isSignedOut && (
              <Burger
                opened={mobileNavOpened}
                onClick={openMobileNav}
                color={theme.other.layout.text}
                size="sm"
                aria-label={
                  mobileNavOpened
                    ? t("header.closeNavigation")
                    : t("header.openNavigation")
                }
              />
            )}
          </Group>
        </Group>
      </AppShell.Header>

      {/* Phone: the main menu as a tab bar at the bottom, for signed-in
          members (guests and workshop mode keep their own menus) */}
      {forceMobile && !isSignedOut && !isGuest && !isInWorkshopMode && <MobileTabBar />}

      {/* Mobile Navigation Drawer */}
      <Drawer
        opened={mobileNavOpened}
        onClose={closeMobileNav}
        size="100%"
        withCloseButton
        styles={{
          header: {
            background: theme.other.layout.headerGradient,
            borderBottom: theme.other.layout.borderLight,
          },
          close: {
            color: theme.other.layout.text,
            "&:hover": {
              backgroundColor: theme.other.layout.bgHover,
            },
          },
          content: {
            background: theme.other.layout.drawerGradient,
            display: "flex",
            flexDirection: "column",
            height: "100%",
          },
          body: {
            padding: 0,
            flex: 1,
            display: "flex",
            flexDirection: "column",
            minHeight: 0,
          },
        }}
        title={
          <Image
            src={logoLandscape}
            alt={branding.siteName}
            h={40}
            w="auto"
            fit="contain"
          />
        }
      >
        {isGuest ? (
          <GuestMobileNavigation
            items={navItems}
            onClose={closeMobileNav}
          />
        ) : isInWorkshopMode ? (
          <WorkshopModeMobileNavigation
            items={navItems}
            onExitWorkshopMode={onExitWorkshopMode}
            onClose={closeMobileNav}
          />
        ) : (
          <MobileNavigation
            items={navItems}
            onSettingsClick={onSettingsClick}
            onProfileClick={onProfileClick}
            onApiKeysClick={onApiKeysClick}
            onLogoutClick={onLogoutClick}
            onClose={closeMobileNav}
          />
        )}
      </Drawer>
    </>
  );
}
