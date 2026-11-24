import {
  Group,
  UnstyledButton,
  Text,
  Tooltip,
} from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../../../providers/AuthProvider";
import {
  IconSettings,
  IconUser,
  IconLogout,
  IconKey,
  IconMail,
} from "@tabler/icons-react";
import { LanguageSwitcher } from "../../LanguageSwitcher";
import { DropdownMenu } from "../../DropdownMenu";
import navClasses from "./DesktopNavigation.module.css";
import { UserAvatar } from "../../UserAvatar";
import { NotificationBell } from "../../NotificationBell";
import { GuestUserMenu } from "../GuestUserMenu";
import { EXTERNAL_LINKS } from "../../../../config/externalLinks";
import type { AppHeaderProps } from "./types";

export function UserActions({
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
  isGuest = false,
  isSignedOut = false,
  isInWorkshopMode = false,
  onExitWorkshopMode,
}: Pick<
  AppHeaderProps,
  | "onSettingsClick"
  | "onProfileClick"
  | "onApiKeysClick"
  | "onLogoutClick"
  | "isGuest"
  | "isSignedOut"
  | "isInWorkshopMode"
  | "workshopName"
  | "onExitWorkshopMode"
>) {
  const { t } = useTranslation("common");
  const { logout: authLogout, backendUser } = useAuth();

  const handleLogout = () => {
    authLogout();
    onLogoutClick?.();
  };

  // Signed out: Contact + Language switcher only, no user menu (signed-in users find Contact in the footer)
  if (isSignedOut) {
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
            className={navClasses.navButton}
          >
            <IconMail size={18} />
            <Text size="sm" fw={600}>
              {t("header.contact")}
            </Text>
          </UnstyledButton>
        </Tooltip>
        <LanguageSwitcher size="sm" variant="compact" />
      </Group>
    );
  }

  const userAvatar = (
    <UnstyledButton aria-label={t("header.openUserMenu")} style={{ borderRadius: 999 }}>
      <UserAvatar name={backendUser?.name || "User"} seed={backendUser?.id} size="md" ring />
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

  // For staff/head in workshop mode: Leave Workshop-Mode button (red) + full user avatar menu
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
            className={navClasses.navButton}
          >
            <IconLogout size={18} />
            <Text size="sm" fw={600}>
              {t("header.leaveWorkshopMode")}
            </Text>
          </UnstyledButton>
        </Tooltip>
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

  // For a Guest in a workshop, show simplified menu
  if (isGuest) {
    return (
      <Group gap="sm" wrap="nowrap">
        <LanguageSwitcher size="sm" variant="compact" />
        <GuestUserMenu
          workshopName={backendUser?.role?.workshop?.name}
          organizationName={backendUser?.role?.institution?.name}
        />
      </Group>
    );
  }

  return (
    <Group gap="sm" wrap="nowrap">
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
