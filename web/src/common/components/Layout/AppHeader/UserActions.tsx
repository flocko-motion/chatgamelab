import {
  Group,
  Box,
  UnstyledButton,
  Text,
  useMantineTheme,
  Tooltip,
} from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../../../providers/AuthProvider";
import {
  IconSettings,
  IconUser,
  IconLogout,
  IconKey,
  IconMessage,
} from "@tabler/icons-react";
import { LanguageSwitcher } from "../../LanguageSwitcher";
import { DropdownMenu } from "../../DropdownMenu";
import { UserAvatar } from "../../UserAvatar";
import { NotificationBell } from "../../NotificationBell";
import { ParticipantUserMenu } from "../ParticipantUserMenu";
import { getUserAvatarColor } from "@/common/lib/userUtils";
import { EXTERNAL_LINKS } from "../../../../config/externalLinks";
import type { AppHeaderProps } from "./types";

export function UserActions({
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
