import {
  AppShell,
  Group,
  Box,
  Burger,
  Drawer,
  Stack,
  Divider,
  Image,
  UnstyledButton,
  Text,
  useMantineTheme,
  Tooltip,
} from "@mantine/core";
import { useDisclosure, useElementSize } from "@mantine/hooks";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { useResponsiveDesign } from "../../hooks/useResponsiveDesign";
import { useAuth } from "../../../providers/AuthProvider";
import {
  IconSettings,
  IconUser,
  IconLogout,
  IconKey,
  IconMessage,
  IconChevronDown,
} from "@tabler/icons-react";
import { LanguageSwitcher } from "../LanguageSwitcher";
import { VersionDisplay } from "../VersionDisplay";
import { DropdownMenu } from "../DropdownMenu";
import { UserAvatar } from "../UserAvatar";
import { NotificationBell } from "../NotificationBell";
import { ParticipantUserMenu } from "./ParticipantUserMenu";
import { getUserAvatarColor } from "@/common/lib/userUtils";
import { ROUTES } from "../../routes/routes";
import { EXTERNAL_LINKS } from "../../../config/externalLinks";
import logoLandscapeWhite from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Landscape-Colorful-White-Text-Transparent.png";

export interface NavItem {
  label: string;
  icon: React.ReactNode;
  onClick?: () => void;
  href?: string;
  active?: boolean;
  children?: NavItem[];
}

export interface AppHeaderProps {
  navItems?: NavItem[];
  onSettingsClick?: () => void;
  onProfileClick?: () => void;
  onApiKeysClick?: () => void;
  onLogoutClick?: () => void;
  userName?: string;
  /** If true, shows simplified participant UI (anonymous participant) */
  isParticipant?: boolean;
  /** If true, staff/head has entered workshop mode (keep user bubble, show exit button) */
  isInWorkshopMode?: boolean;
  /** Name of the workshop when in workshop mode */
  workshopName?: string | null;
  /** Callback to exit workshop mode */
  onExitWorkshopMode?: () => void;
}

function NavButton({ item }: { item: NavItem }) {
  const theme = useMantineTheme();

  return (
    <UnstyledButton
      onClick={item.onClick}
      py="xs"
      px="md"
      style={{
        borderRadius: "var(--mantine-radius-md)",
        color: "white",
        display: "flex",
        alignItems: "center",
        gap: "8px",
        transition: "background-color 150ms ease, box-shadow 150ms ease",
      }}
      styles={{
        root: {
          backgroundColor: item.active
            ? theme.other.layout.bgActive
            : "transparent",
          boxShadow: item.active
            ? "0 0 0 1px rgba(255, 255, 255, 0.3)"
            : "none",
          "&:hover": {
            backgroundColor: item.active
              ? theme.other.layout.bgActive
              : "rgba(255, 255, 255, 0.2)",
            boxShadow: item.active
              ? "0 0 0 1px rgba(255, 255, 255, 0.3)"
              : "none",
          },
        },
      }}
    >
      {item.icon}
      <Text size="sm" fw={500}>
        {item.label}
      </Text>
    </UnstyledButton>
  );
}

function NavDropdownButton({ item }: { item: NavItem }) {
  const theme = useMantineTheme();
  const hasActiveChild = item.children?.some((child) => child.active);
  const isActive = item.active || hasActiveChild;

  const trigger = (
    <UnstyledButton
      py="xs"
      px="md"
      style={{
        borderRadius: "var(--mantine-radius-md)",
        color: "white",
        display: "flex",
        alignItems: "center",
        gap: "8px",
        transition: "background-color 150ms ease, box-shadow 150ms ease",
      }}
      styles={{
        root: {
          backgroundColor: isActive
            ? theme.other.layout.bgActive
            : "transparent",
          boxShadow: isActive ? "0 0 0 1px rgba(255, 255, 255, 0.3)" : "none",
          "&:hover": {
            backgroundColor: isActive
              ? theme.other.layout.bgActive
              : "rgba(255, 255, 255, 0.2)",
          },
        },
      }}
    >
      {item.icon}
      <Text size="sm" fw={500}>
        {item.label}
      </Text>
      <IconChevronDown size={14} style={{ opacity: 0.7 }} />
    </UnstyledButton>
  );

  const menuItems =
    item.children?.map((child, idx) => ({
      key: `${child.label}-${idx}`,
      label: child.label,
      icon: child.icon,
      onClick: child.onClick,
    })) || [];

  return <DropdownMenu trigger={trigger} items={menuItems} position="bottom" />;
}

function DesktopNavigation({ items }: { items: NavItem[] }) {
  return (
    <Group gap="xs" wrap="nowrap">
      {items.map((item, index) =>
        item.children ? (
          <NavDropdownButton key={index} item={item} />
        ) : (
          <NavButton key={index} item={item} />
        ),
      )}
    </Group>
  );
}

function UserActions({
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
  isParticipant = false,
  isInWorkshopMode = false,
  workshopName: _workshopName,
  onExitWorkshopMode,
}: Pick<
  AppHeaderProps,
  | "onSettingsClick"
  | "onProfileClick"
  | "onApiKeysClick"
  | "onLogoutClick"
  | "isParticipant"
  | "isInWorkshopMode"
  | "workshopName"
  | "onExitWorkshopMode"
>) {
  const { t } = useTranslation("common");
  const { logout: authLogout, backendUser } = useAuth();
  const theme = useMantineTheme();

  const handleLogout = () => {
    authLogout();
    onLogoutClick?.();
  };

  const userAvatar = (
    <UnstyledButton
      aria-label={t("header.openUserMenu")}
      style={{
        borderRadius: 999,
        padding: 2,
        border: `2px solid ${theme.colors[getUserAvatarColor(backendUser?.name || "User")]?.[6] || theme.colors.accent[6]}`,
        backgroundColor: theme.other.layout.bgSubtle,
        transition: "background-color 150ms ease, border-color 150ms ease",
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
      <UserAvatar
        name={backendUser?.name || "User"}
        size="md"
        style={{
          backgroundColor: "transparent",
        }}
      />
    </UnstyledButton>
  );

  const userMenuItems = [
    {
      key: "profile",
      label: t("profile"),
      icon: <IconUser size={16} />,
      onClick: onProfileClick,
    },
    {
      key: "settings",
      label: t("settings"),
      icon: <IconSettings size={16} />,
      onClick: onSettingsClick,
    },
    {
      key: "apiKeys",
      label: t("header.apiKeys"),
      icon: <IconKey size={16} />,
      onClick: onApiKeysClick,
    },
    {
      key: "logout",
      label: t("logout"),
      icon: <IconLogout size={16} />,
      onClick: handleLogout,
      danger: true,
    },
  ];

  // For staff/head in workshop mode: Leave Workshop-Mode button (red) + Contact + full user avatar menu
  if (isInWorkshopMode && onExitWorkshopMode) {
    return (
      <Group gap="sm" wrap="nowrap">
        <Tooltip
          label={t("header.leaveWorkshopModeTooltip")}
          position="bottom"
          withArrow
        >
          <UnstyledButton
            onClick={onExitWorkshopMode}
            aria-label={t("header.leaveWorkshopModeTooltip")}
            py="xs"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
              transition: "background-color 150ms ease, box-shadow 150ms ease",
            }}
            styles={{
              root: {
                backgroundColor: theme.colors.red[6],
                boxShadow: "none",
                "&:hover": {
                  backgroundColor: theme.colors.red[7],
                  boxShadow: "none",
                },
              },
            }}
          >
            <IconLogout size={18} />
            <Text size="sm" fw={500}>
              {t("header.leaveWorkshopMode")}
            </Text>
          </UnstyledButton>
        </Tooltip>
        <Tooltip
          label={EXTERNAL_LINKS.CONTACT.description}
          position="bottom"
          withArrow
        >
          <UnstyledButton
            onClick={() => window.open(EXTERNAL_LINKS.CONTACT.href, "_blank")}
            aria-label={t("header.contact")}
            py="xs"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
              transition: "background-color 150ms ease, box-shadow 150ms ease",
            }}
            styles={{
              root: {
                backgroundColor: "transparent",
                boxShadow: "none",
                "&:hover": {
                  backgroundColor: "rgba(255, 255, 255, 0.2)",
                  boxShadow: "none",
                },
              },
            }}
          >
            <IconMessage size={18} />
            <Text size="sm" fw={500}>
              {t("header.contact")}
            </Text>
          </UnstyledButton>
        </Tooltip>
        <Box w="lg" />
        <NotificationBell />
        <LanguageSwitcher size="sm" variant="compact" />
        <DropdownMenu
          trigger={userAvatar}
          items={userMenuItems}
          position="bottom"
        />
      </Group>
    );
  }

  // For participants, show simplified menu
  if (isParticipant) {
    return (
      <Group gap="sm" wrap="nowrap">
        <Tooltip
          label={EXTERNAL_LINKS.CONTACT.description}
          position="bottom"
          withArrow
        >
          <UnstyledButton
            onClick={() => window.open(EXTERNAL_LINKS.CONTACT.href, "_blank")}
            aria-label={t("header.contact")}
            py="xs"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
              transition: "background-color 150ms ease, box-shadow 150ms ease",
            }}
            styles={{
              root: {
                backgroundColor: "transparent",
                boxShadow: "none",
                "&:hover": {
                  backgroundColor: "rgba(255, 255, 255, 0.2)",
                  boxShadow: "none",
                },
              },
            }}
          >
            <IconMessage size={18} />
            <Text size="sm" fw={500}>
              {t("header.contact")}
            </Text>
          </UnstyledButton>
        </Tooltip>
        <LanguageSwitcher size="sm" variant="compact" />
        <ParticipantUserMenu
          workshopName={backendUser?.role?.workshop?.name}
          organizationName={backendUser?.role?.institution?.name}
        />
      </Group>
    );
  }

  return (
    <Group gap="sm" wrap="nowrap">
      <Tooltip
        label={EXTERNAL_LINKS.CONTACT.description}
        position="bottom"
        withArrow
      >
        <UnstyledButton
          onClick={() => window.open(EXTERNAL_LINKS.CONTACT.href, "_blank")}
          aria-label={t("header.contact")}
          py="xs"
          px="md"
          style={{
            borderRadius: "var(--mantine-radius-md)",
            color: "white",
            display: "flex",
            alignItems: "center",
            gap: "8px",
            transition: "background-color 150ms ease, box-shadow 150ms ease",
          }}
          styles={{
            root: {
              backgroundColor: "transparent",
              boxShadow: "none",
              "&:hover": {
                backgroundColor: "rgba(255, 255, 255, 0.2)",
                boxShadow: "none",
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
          <IconMessage size={18} />
          <Text size="sm" fw={500}>
            {t("header.contact")}
          </Text>
        </UnstyledButton>
      </Tooltip>
      <Box w="lg" />
      <NotificationBell />
      <LanguageSwitcher size="sm" variant="compact" />
      <DropdownMenu
        trigger={userAvatar}
        items={userMenuItems}
        position="bottom"
      />
    </Group>
  );
}

/**
 * Simplified mobile navigation for workshop participants.
 * Shows only: My Workshop, Contact, Language, Workshop info, Leave Workshop
 */
function ParticipantMobileNavigation({
  items,
  onClose,
}: {
  items: NavItem[];
  onClose: () => void;
}) {
  const { t } = useTranslation("common");
  const { t: tParticipant } = useTranslation("participant");
  const { logout: authLogout, backendUser } = useAuth();
  const theme = useMantineTheme();

  const handleLeaveWorkshop = () => {
    authLogout();
    onClose();
  };

  const workshopName = backendUser?.role?.workshop?.name;
  const organizationName = backendUser?.role?.institution?.name;

  return (
    <Box style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <Box style={{ flex: 1, overflowY: "auto" }}>
        <Stack gap={4} p="sm">
          {/* My Workshop nav item */}
          {items.map((item, index) => (
            <UnstyledButton
              key={index}
              onClick={() => {
                item.onClick?.();
                onClose();
              }}
              py="sm"
              px="md"
              style={{
                borderRadius: "var(--mantine-radius-md)",
                color: "white",
                display: "flex",
                alignItems: "center",
                gap: "10px",
                backgroundColor: item.active
                  ? theme.other.layout.bgActive
                  : theme.other.layout.bgSubtle,
                boxShadow: item.active
                  ? "0 0 0 1px rgba(255, 255, 255, 0.3)"
                  : "none",
              }}
            >
              {item.icon}
              <Text size="md" fw={500}>
                {item.label}
              </Text>
            </UnstyledButton>
          ))}
        </Stack>
      </Box>

      <Box p="sm">
        <Divider my="xs" color={theme.other.layout.lineLight} />

        {/* Workshop and Organization info */}
        <Stack gap="xs" px="md" py="sm">
          {workshopName && (
            <Group gap="xs">
              <Text size="xs" c="dimmed">
                {tParticipant("workshop")}:
              </Text>
              <Text size="sm" fw={500} c="white">
                {workshopName}
              </Text>
            </Group>
          )}
          {organizationName && (
            <Group gap="xs">
              <Text size="xs" c="dimmed">
                {tParticipant("organization")}:
              </Text>
              <Text size="sm" fw={500} c="white">
                {organizationName}
              </Text>
            </Group>
          )}
        </Stack>

        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Stack gap={4}>
          {/* Contact */}
          <UnstyledButton
            onClick={() => {
              window.open(EXTERNAL_LINKS.CONTACT.href, "_blank");
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconMessage size={18} />
            <Text size="sm" fw={500}>
              {t("header.contact")}
            </Text>
          </UnstyledButton>

          {/* Leave Workshop */}
          <UnstyledButton
            onClick={handleLeaveWorkshop}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: theme.colors.red[6],
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconLogout size={18} />
            <Text size="sm" fw={500}>
              {tParticipant("leaveWorkshop")}
            </Text>
          </UnstyledButton>
        </Stack>

        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Group px="lg" justify="space-between">
          <VersionDisplay darkMode />
          <LanguageSwitcher size="sm" variant="compact" />
        </Group>
      </Box>
    </Box>
  );
}

function MobileNavigation({
  items,
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
  onClose,
}: {
  items: NavItem[];
  onSettingsClick?: () => void;
  onProfileClick?: () => void;
  onApiKeysClick?: () => void;
  onLogoutClick?: () => void;
  onClose: () => void;
}) {
  const { t } = useTranslation("common");
  const { logout: authLogout } = useAuth();
  const theme = useMantineTheme();

  const handleLogout = () => {
    authLogout();
    onLogoutClick?.();
    onClose();
  };

  // Flatten items with children for mobile view
  const flattenedItems: (NavItem & {
    isChild?: boolean;
    parentLabel?: string;
  })[] = [];
  items.forEach((item) => {
    if (item.children && item.children.length > 0) {
      // Add parent as a label/header, not clickable
      flattenedItems.push({ ...item, onClick: undefined });
      // Add children as indented items
      item.children.forEach((child) => {
        flattenedItems.push({
          ...child,
          isChild: true,
          parentLabel: item.label,
        });
      });
    } else {
      flattenedItems.push(item);
    }
  });

  return (
    <Box style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <Box style={{ flex: 1, overflowY: "auto" }}>
        <Stack gap={4} p="sm">
          {flattenedItems.map((item, index) => {
            const isParentWithChildren =
              item.children && item.children.length > 0;
            const isChild = "isChild" in item && item.isChild;

            // Parent items with children are rendered as non-clickable headers
            if (isParentWithChildren) {
              return (
                <Text
                  key={index}
                  size="xs"
                  fw={600}
                  c="dimmed"
                  px="md"
                  pt="sm"
                  pb={4}
                  style={{ textTransform: "uppercase", letterSpacing: "0.5px" }}
                >
                  {item.label}
                </Text>
              );
            }

            return (
              <UnstyledButton
                key={index}
                onClick={() => {
                  item.onClick?.();
                  onClose();
                }}
                py="sm"
                px="md"
                pl={isChild ? "lg" : "md"}
                style={{
                  borderRadius: "var(--mantine-radius-md)",
                  color: "white",
                  display: "flex",
                  alignItems: "center",
                  gap: "10px",
                  transition:
                    "background-color 150ms ease, box-shadow 150ms ease",
                  backgroundColor: item.active
                    ? theme.other.layout.bgActive
                    : theme.other.layout.bgSubtle,
                  boxShadow: item.active
                    ? "0 0 0 1px rgba(255, 255, 255, 0.3)"
                    : "none",
                  marginLeft: isChild ? "12px" : 0,
                }}
              >
                {item.icon}
                <Text size={isChild ? "sm" : "md"} fw={500}>
                  {item.label}
                </Text>
              </UnstyledButton>
            );
          })}
        </Stack>
      </Box>

      <Box p="sm">
        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Stack gap={4}>
          <UnstyledButton
            onClick={() => {
              window.open(EXTERNAL_LINKS.CHATGAMELAB.href, "_blank");
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconMessage size={18} />
            <Text size="sm" fw={500}>
              {t("header.contact")}
            </Text>
          </UnstyledButton>

          <UnstyledButton
            onClick={() => {
              onProfileClick?.();
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconUser size={18} />
            <Text size="sm" fw={500}>
              {t("header.profile")}
            </Text>
          </UnstyledButton>

          <UnstyledButton
            onClick={() => {
              onSettingsClick?.();
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconSettings size={18} />
            <Text size="sm" fw={500}>
              {t("settings")}
            </Text>
          </UnstyledButton>

          <UnstyledButton
            onClick={() => {
              onApiKeysClick?.();
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: "white",
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconKey size={18} />
            <Text size="sm" fw={500}>
              {t("header.apiKeys")}
            </Text>
          </UnstyledButton>

          <UnstyledButton
            onClick={handleLogout}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: theme.colors.red[6],
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconLogout size={18} />
            <Text size="sm" fw={500}>
              {t("header.logout")}
            </Text>
          </UnstyledButton>
        </Stack>

        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Group px="lg" justify="space-between">
          <VersionDisplay darkMode />
          <LanguageSwitcher size="sm" variant="compact" />
        </Group>
      </Box>
    </Box>
  );
}

export function AppHeader({
  navItems = [],
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
  isParticipant = false,
  isInWorkshopMode = false,
  workshopName,
  onExitWorkshopMode,
}: AppHeaderProps) {
  const [mobileNavOpened, { open: openMobileNav, close: closeMobileNav }] =
    useDisclosure(false);
  const { isMobile: isViewportMobile } = useResponsiveDesign();
  const { t } = useTranslation("common");
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
            src={logoLandscapeWhite}
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
            isParticipant={isParticipant}
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
            {/* Logo */}
            <UnstyledButton
              onClick={() => navigate({ to: ROUTES.DASHBOARD })}
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
                src={logoLandscapeWhite}
                alt="ChatGameLab Logo"
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
            {/* Desktop: Full user actions */}
            {!forceMobile && (
              <UserActions
                onSettingsClick={onSettingsClick}
                onProfileClick={onProfileClick}
                onApiKeysClick={onApiKeysClick}
                onLogoutClick={onLogoutClick}
                isParticipant={isParticipant}
                isInWorkshopMode={isInWorkshopMode}
                workshopName={workshopName}
                onExitWorkshopMode={onExitWorkshopMode}
              />
            )}

            {/* Mobile: Burger menu */}
            {forceMobile && (
              <Burger
                opened={mobileNavOpened}
                onClick={openMobileNav}
                color="white"
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
            color: "white",
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
            src={logoLandscapeWhite}
            alt="ChatGameLab Logo"
            h={40}
            w="auto"
            fit="contain"
          />
        }
      >
        {isParticipant ? (
          <ParticipantMobileNavigation
            items={navItems}
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
