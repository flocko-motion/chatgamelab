import {
  Card,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
  ThemeIcon,
  Badge,
} from '@mantine/core';
import {
  IconBuilding,
  IconChartBar,
  IconDeviceGamepad2,
  IconMessage,
  IconUsers,
  IconPlayerPlay,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useAuth } from '@/providers/AuthProvider';
import { UserAvatar } from '@/common/components/UserAvatar';

export function ProfileView() {
  const { t } = useTranslation('auth');
  const { backendUser } = useAuth();

  if (!backendUser) {
    return null;
  }

  // Get translated role name
  const getRoleTranslation = (role: string) => {
    return t(`profile.roles.${role}`);
  };

  // Format member since date
  const memberSince = backendUser.meta?.createdAt
    ? new Date(backendUser.meta.createdAt).toLocaleDateString()
    : '-';

  // Dummy statistics for now
  const stats = [
    { label: t('profile.gamesPlayed'), value: 12, icon: IconPlayerPlay },
    { label: t('profile.gamesCreated'), value: 3, icon: IconDeviceGamepad2 },
    { label: t('profile.messagesSent'), value: 156, icon: IconMessage },
    { label: t('profile.sessionsJoined'), value: 8, icon: IconUsers },
  ];

  return (
    <Stack gap="xl">
      {/* User Info Card */}
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Group gap="lg" align="flex-start">
          <UserAvatar
            name={backendUser.name || 'User'}
            size="xl"
          />
          
          <Stack gap="xs" style={{ flex: 1 }}>
            <Title order={2}>{backendUser.name}</Title>
            {backendUser.email && (
              <Text c="dimmed" size="sm">{backendUser.email}</Text>
            )}
            <Text size="sm" c="dimmed">
              {t('profile.memberSince')}: {memberSince}
            </Text>
          </Stack>
        </Group>
      </Card>

      {/* Organization Card */}
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack gap="md">
          <Group gap="sm">
            <ThemeIcon variant="light" size="lg" color="accent">
              <IconBuilding size={20} />
            </ThemeIcon>
            <Title order={3}>{t('profile.organizationSection')}</Title>
          </Group>
          
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            <Stack gap="xs">
              <Text size="sm" c="dimmed">{t('profile.organizationSection')}</Text>
              <Text fw={500}>
                {backendUser.role?.institution?.name || t('profile.noOrganization')}
              </Text>
            </Stack>
            
            <Stack gap="xs">
              <Text size="sm" c="dimmed">{t('profile.role')}</Text>
              {backendUser.role?.role ? (
                <Badge variant="light" color="accent" size="lg">
                  {getRoleTranslation(backendUser.role.role)}
                </Badge>
              ) : (
                <Text fw={500}>{t('profile.noRole')}</Text>
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
            <Title order={3}>{t('profile.statisticsSection')}</Title>
          </Group>
          
          <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
            {stats.map((stat) => (
              <Card key={stat.label} padding="md" radius="md" bg="gray.0">
                <Stack gap="xs" align="center" ta="center">
                  <ThemeIcon variant="light" size="xl" color="accent" radius="xl">
                    <stat.icon size={24} />
                  </ThemeIcon>
                  <Text size="xl" fw={700}>{stat.value}</Text>
                  <Text size="xs" c="dimmed">{stat.label}</Text>
                </Stack>
              </Card>
            ))}
          </SimpleGrid>
        </Stack>
      </Card>
    </Stack>
  );
}
