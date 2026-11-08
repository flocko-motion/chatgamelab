import {
  Box,
  Stack,
  Divider,
  Group,
  UnstyledButton,
  Text,
  useMantineTheme,
} from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../../../../providers/AuthProvider";
import {
  IconSettings,
  IconUser,
  IconLogout,
  IconKey,
  IconMessage,
} from "@tabler/icons-react";
import { LanguageSwitcher } from "../../../LanguageSwitcher";
import { VersionDisplay } from "../../../VersionDisplay";
import { EXTERNAL_LINKS } from "../../../../../config/externalLinks";
import type { NavItem } from "../types";

export interface MobileNavigationProps {
  items: NavItem[];
  onSettingsClick?: () => void;
  onProfileClick?: () => void;
  onApiKeysClick?: () => void;
  onLogoutClick?: () => void;
  onClose: () => void;
}

/**
 * Standard mobile navigation for regular authenticated users.
 * Shows all nav items, user actions, and logout.
 */
export function MobileNavigation({
  items,
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
  onClose,
}: MobileNavigationProps) {
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
