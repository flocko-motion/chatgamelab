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
import { IconLogout, IconMessage } from "@tabler/icons-react";
import { LanguageSwitcher } from "../../../LanguageSwitcher";
import { VersionDisplay } from "../../../VersionDisplay";
import { EXTERNAL_LINKS } from "../../../../../config/externalLinks";
import type { NavItem } from "../types";

export interface WorkshopModeMobileNavigationProps {
  items: NavItem[];
  onExitWorkshopMode?: () => void;
  onClose: () => void;
}

/**
 * Simplified mobile navigation for staff/head/individual users in workshop mode.
 * Shows workshop nav items, workshop info, contact, and exit workshop mode.
 * Similar to participant view but with "Leave Workshop Mode" instead of logout.
 */
export function WorkshopModeMobileNavigation({
  items,
  onExitWorkshopMode,
  onClose,
}: WorkshopModeMobileNavigationProps) {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const theme = useMantineTheme();

  const handleExitWorkshopMode = () => {
    onExitWorkshopMode?.();
    onClose();
  };

  const workshopName = backendUser?.role?.workshop?.name;
  const organizationName = backendUser?.role?.institution?.name;

  return (
    <Box style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <Box style={{ flex: 1, overflowY: "auto" }}>
        <Stack gap={4} p="sm">
          {/* Workshop nav items (My Workshop, Workshop Settings) */}
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
                {t("header.workshop")}:
              </Text>
              <Text size="sm" fw={500} c="white">
                {workshopName}
              </Text>
            </Group>
          )}
          {organizationName && (
            <Group gap="xs">
              <Text size="xs" c="dimmed">
                {t("header.organization")}:
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

          {/* Leave Workshop Mode */}
          <UnstyledButton
            onClick={handleExitWorkshopMode}
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
              {t("header.leaveWorkshopMode")}
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
