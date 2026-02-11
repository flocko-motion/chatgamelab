import {
  Card,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
  ThemeIcon,
  Badge,
  Skeleton,
} from "@mantine/core";
import {
  IconBuilding,
  IconChartBar,
  IconDeviceGamepad2,
  IconMessage,
  IconPlayerPlay,
  IconShieldStar,
  IconTrophy,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { useAuth } from "@/providers/AuthProvider";
import { UserAvatar } from "@/common/components/UserAvatar";
import { useUserStats } from "@/api/hooks";
import {
  isAdmin,
  getUserRole,
  getRoleColor,
  useTranslateRole,
} from "@/common/lib/roles";

export function ProfileView() {
  const { t } = useTranslation("auth");
  const { backendUser } = useAuth();
  const { data: stats, isLoading: statsLoading } = useUserStats();

  const translateRole = useTranslateRole(t("profile.noRole"));

  if (!backendUser) {
    return null;
  }

  const userIsAdmin = isAdmin(backendUser);
  const userRole = getUserRole(backendUser);
  const hasOrganization = !!backendUser.role?.institution;

  // Format member since date
  const memberSince = backendUser.meta?.createdAt
    ? new Date(backendUser.meta.createdAt).toLocaleDateString()
    : "-";

  // Statistics from API
  const statItems = [
    {
      label: t("profile.gamesPlayed"),
      value: stats?.gamesPlayed ?? 0,
      icon: IconPlayerPlay,
    },
    {
      label: t("profile.gamesCreated"),
      value: stats?.gamesCreated ?? 0,
      icon: IconDeviceGamepad2,
    },
    {
      label: t("profile.messagesSent"),
      value: stats?.messagesSent ?? 0,
      icon: IconMessage,
    },
    {
      label: t("profile.totalPlaysOnGames"),
      value: stats?.totalPlaysOnGames ?? 0,
      icon: IconTrophy,
    },
  ];

  return (
    <Stack gap="xl">
      {/* User Info Card */}
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Group gap="lg" align="flex-start">
          <UserAvatar name={backendUser.name || "User"} size="xl" />

          <Stack gap="xs" style={{ flex: 1 }}>
            <Group gap="sm" align="center">
              <Title order={2}>{backendUser.name}</Title>
              {userIsAdmin && (
                <Badge
                  variant="filled"
                  color="red"
                  size="lg"
                  leftSection={<IconShieldStar size={14} />}
                >
                  Admin
                </Badge>
              )}
            </Group>
            {backendUser.email && (
              <Text c="dimmed" size="sm">
                {backendUser.email}
              </Text>
            )}
            <Text size="sm" c="dimmed">
              {t("profile.memberSince")}: {memberSince}
            </Text>
          </Stack>
        </Group>
      </Card>

      {/* Organization & Role Card */}
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack gap="md">
          <Group gap="sm">
            <ThemeIcon variant="light" size="lg" color="accent">
              <IconBuilding size={20} />
            </ThemeIcon>
            <Title order={3}>{t("profile.organizationSection")}</Title>
          </Group>

          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            <Stack gap="xs">
              <Text size="sm" c="dimmed">
                {t("profile.organization")}
              </Text>
              <Text fw={500}>
                {hasOrganization
                  ? backendUser.role?.institution?.name
                  : t("profile.noOrganization")}
              </Text>
            </Stack>

            <Stack gap="xs">
              <Text size="sm" c="dimmed">
                {t("profile.role")}
              </Text>
              {userRole !== undefined ? (
                <Badge
                  variant={userIsAdmin ? "filled" : "light"}
                  color={getRoleColor(backendUser.role?.role)}
                  size="lg"
                >
                  {translateRole(backendUser.role?.role)}
                </Badge>
              ) : (
                <Text fw={500}>{t("profile.noRole")}</Text>
              )}
            </Stack>
          </SimpleGrid>
        </Stack>
      </Card>

      {/* Statistics Card */}
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack gap="md">
          <Group gap="sm">
            <ThemeIcon variant="light" size="lg" color="accent">
              <IconChartBar size={20} />
            </ThemeIcon>
            <Title order={3}>{t("profile.statisticsSection")}</Title>
          </Group>

          <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
            {statItems.map((stat) => (
              <Card key={stat.label} padding="md" radius="md" bg="gray.0">
                <Stack gap="xs" align="center" ta="center">
                  <ThemeIcon
                    variant="light"
                    size="xl"
                    color="accent"
                    radius="xl"
                  >
                    <stat.icon size={24} />
                  </ThemeIcon>
                  {statsLoading ? (
                    <Skeleton height={28} width={40} />
                  ) : (
                    <Text size="xl" fw={700}>
                      {stat.value}
                    </Text>
                  )}
                  <Text size="xs" c="dimmed">
                    {stat.label}
                  </Text>
                </Stack>
              </Card>
            ))}
          </SimpleGrid>
        </Stack>
      </Card>
    </Stack>
  );
}
