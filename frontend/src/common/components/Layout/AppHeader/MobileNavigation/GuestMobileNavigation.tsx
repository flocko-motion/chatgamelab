import {
  Box,
  Stack,
  Divider,
  Group,
  UnstyledButton,
  Text,
  useMantineTheme,
} from "@mantine/core";
import { useBranding } from "@/providers/branding";
import { useModals } from "@mantine/modals";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../../../../providers/AuthProvider";
import { IconMail, IconLogout, IconUser } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { ROUTES } from "@/common/routes/routes";
import { LanguageSwitcher } from "../../../LanguageSwitcher";
import { VersionDisplay } from "../../../VersionDisplay";
import type { NavItem } from "../types";

export interface GuestMobileNavigationProps {
  items: NavItem[];
  onClose: () => void;
}

/**
 * Simplified mobile navigation for a Guest in a workshop.
 * Shows only: My Workshop, Contact, Profile, Language, Workshop info, Leave Workshop
 */
export function GuestMobileNavigation({
  items,
  onClose,
}: GuestMobileNavigationProps) {  
  const { t } = useTranslation("common");
  const branding = useBranding();
  const { t: tAuth } = useTranslation("auth");
  const { logout: authLogout, backendUser } = useAuth();
  const theme = useMantineTheme();
  const modals = useModals();
  const navigate = useNavigate();

  const handleLeaveWorkshop = () => {
    modals.openConfirmModal({
      title: tAuth("guest.leaveConfirm.title"),
      children: (
        <Text size="sm">
          {tAuth("guest.leaveConfirm.message")}
        </Text>
      ),
      labels: {
        confirm: tAuth("guest.leaveConfirm.confirm"),
        cancel: t("cancel"),
      },
      confirmProps: { color: "red" },
      onConfirm: async () => {
        onClose();
        await authLogout();
      },
    });
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
                color: theme.other.layout.text,
                display: "flex",
                alignItems: "center",
                gap: "10px",
                backgroundColor: item.active
                  ? theme.other.layout.bgActive
                  : theme.other.layout.bgSubtle,
                boxShadow: item.active
                  ? "inset 0 0 0 2px var(--mantine-color-ink-5)"
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
                {tAuth("guest.workshop")}:
              </Text>
              <Text size="sm" fw={500} c={theme.other.layout.text}>
                {workshopName}
              </Text>
            </Group>
          )}
          {organizationName && (
            <Group gap="xs">
              <Text size="xs" c="dimmed">
                {tAuth("guest.organization")}:
              </Text>
              <Text size="sm" fw={500} c={theme.other.layout.text}>
                {organizationName}
              </Text>
            </Group>
          )}
        </Stack>

        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Stack gap={4}>
          {/* Contact */}
          {branding.contactUrl && (
          <UnstyledButton
            onClick={() => {
              window.open(branding.contactUrl, "_blank");
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: theme.other.layout.text,
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <IconMail size={18} />
            <Text size="sm" fw={500}>
              {t("header.contact")}
            </Text>
          </UnstyledButton>
          )}

          <UnstyledButton
            onClick={() => {
              navigate({ to: ROUTES.PROFILE });
              onClose();
            }}
            py="sm"
            px="md"
            style={{
              borderRadius: "var(--mantine-radius-md)",
              color: theme.other.layout.text,
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
              {tAuth("guest.leaveWorkshop")}
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
