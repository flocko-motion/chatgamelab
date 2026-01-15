import { 
  Card, 
  Text, 
  Title, 
  Stack, 
  Group, 
  Grid,
  Divider,
  SimpleGrid,
  Box,
  ThemeIcon,
} from '@mantine/core';
import { MenuButton, TextButton } from '@components/buttons';
import { CardTitle, Label, HelperText } from '@components/typography';
import { useTranslation } from 'react-i18next';
import { AppLayout, type NavItem } from '@/common/components/Layout';
import { navigationLogger } from '@/config/logger';
import { 
  IconPlayerPlay, 
  IconEdit, 
  IconBuilding, 
  IconUsers, 
  IconClock,
  IconTrendingUp,
  IconPlus
} from '@tabler/icons-react';

// Quick Stats Cards
function QuickStatsCards() {
  const { t } = useTranslation('dashboard');

  return (
    <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="lg" mb="xl">
      <Card 
        p="xl" 
        withBorder 
        shadow="md"
        style={{ 
          borderTop: '3px solid var(--mantine-color-highlight-5)'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Label uppercase>
              {t('stats.activeAdventures')}
            </Label>
            <Title order={1} c="accent.9" mb="xs">3</Title>
            <HelperText>{t('stats.storiesInProgress', { count: 2 })}</HelperText>
          </div>
          <ThemeIcon 
            size={48} 
            radius="md"
            variant="light"
            color="accent"
          >
            <IconPlayerPlay size={24} />
          </ThemeIcon>
        </Group>
      </Card>

      <Card 
        p="xl" 
        withBorder 
        shadow="md"
        style={{ 
          borderTop: '3px solid var(--mantine-color-blue-5)'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Label uppercase>
              {t('stats.storyDrafts')}
            </Label>
            <Title order={1} c="accent.9" mb="xs">2</Title>
            <HelperText>{t('stats.readyForPlayers')}</HelperText>
          </div>
          <ThemeIcon 
            size={48} 
            radius="md"
            variant="light"
            color="blue"
          >
            <IconEdit size={24} />
          </ThemeIcon>
        </Group>
      </Card>

      <Card 
        p="xl" 
        withBorder 
        shadow="md"
        style={{ 
          borderTop: '3px solid var(--mantine-color-green-5)'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Label uppercase>
              {t('stats.storyRooms')}
            </Label>
            <Title order={1} c="accent.9" mb="xs">5</Title>
            <HelperText>{t('stats.availableSessions')}</HelperText>
          </div>
          <ThemeIcon 
            size={48} 
            radius="md"
            variant="light"
            color="green"
          >
            <IconBuilding size={24} />
          </ThemeIcon>
        </Group>
      </Card>

      <Card 
        p="xl" 
        withBorder 
        shadow="md"
        style={{ 
          borderTop: '3px solid var(--mantine-color-orange-5)'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Label uppercase>
              {t('stats.storytellers')}
            </Label>
            <Title order={1} c="accent.9" mb="xs">12</Title>
            <HelperText>{t('stats.onlineNow')}</HelperText>
          </div>
          <ThemeIcon 
            size={48} 
            radius="md"
            variant="light"
            color="orange"
          >
            <IconUsers size={24} />
          </ThemeIcon>
        </Group>
      </Card>
    </SimpleGrid>
  );
}

// Recent Activity Feed
function RecentActivity() {
  const { t } = useTranslation('dashboard');

  const activities = [
    {
      icon: <IconPlayerPlay size={16} />,
      color: 'accent',
      title: 'Space Adventure',
      user: 'Sarah',
      action: t('recentActivity.continuedPlaying'),
      time: '2 min ago',
      description: 'Chapter 3: The Dark Forest'
    },
    {
      icon: <IconEdit size={16} />,
      color: 'blue',
      title: 'Dragon Quest',
      user: 'Mike',
      action: t('recentActivity.createdNewGame'),
      time: '15 min ago',
      description: 'Fantasy adventure with magic'
    },
    {
      icon: <IconBuilding size={16} />,
      color: 'green',
      title: 'Workshop 3B',
      user: 'Emma',
      action: t('recentActivity.invitedStudents', { count: 5 }),
      time: '1 hour ago',
      description: 'Creative writing session'
    }
  ];

  return (
    <Card p="lg" withBorder shadow="sm">
      <Group justify="space-between" mb="md">
        <CardTitle accent>{t('recentActivity.title')}</CardTitle>
        <TextButton>{t('recentActivity.viewAll')}</TextButton>
      </Group>
      
      <Stack gap="md">
        {activities.map((activity, index) => (
          <Box key={index}>
            <Group gap="sm" align="flex-start">
              <ThemeIcon color={activity.color} size={32} radius="md">
                {activity.icon}
              </ThemeIcon>
              <div style={{ flex: 1 }}>
                <Group gap="xs" align="center" mb={2}>
                  <Text size="sm" fw={600}>{activity.title}</Text>
                  <Text size="xs" c="dimmed">•</Text>
                  <Text size="xs" c="dimmed">{activity.user} {activity.action}</Text>
                </Group>
                <Text size="xs" c="dimmed">{activity.description}</Text>
                <Group gap="xs" mt={4}>
                  <IconClock size={12} color="gray" />
                  <Text size="xs" c="dimmed">{activity.time}</Text>
                </Group>
              </div>
            </Group>
            {index < activities.length - 1 && <Divider my="sm" />}
          </Box>
        ))}
      </Stack>
    </Card>
  );
}

// Quick Actions
function QuickActions() {
  const { t } = useTranslation('dashboard');

  return (
    <Card p="lg" withBorder shadow="sm">
      <CardTitle accent>{t('quickActions.title')}</CardTitle>
      <Stack gap="sm">
        <MenuButton leftSection={<IconPlus size={16} />}>
          {t('quickActions.createNewGame')}
        </MenuButton>
        <MenuButton leftSection={<IconBuilding size={16} />}>
          {t('quickActions.createRoom')}
        </MenuButton>
        <MenuButton leftSection={<IconUsers size={16} />}>
          {t('quickActions.inviteMembers')}
        </MenuButton>
        <MenuButton leftSection={<IconTrendingUp size={16} />}>
          {t('quickActions.viewAnalytics')}
        </MenuButton>
      </Stack>
    </Card>
  );
}

/**
 * Dashboard content component - can be used standalone or within AppLayout
 */
export function DashboardContent() {
  return (
    <Stack gap="xl">
      <QuickStatsCards />
      
      <Grid>
        <Grid.Col span={{ base: 12, lg: 8 }}>
          <RecentActivity />
        </Grid.Col>
        <Grid.Col span={{ base: 12, lg: 4 }}>
          <QuickActions />
        </Grid.Col>
      </Grid>
    </Stack>
  );
}

/**
 * Full Dashboard page with layout - for standalone use
 */
export function Dashboard() {
  const { t } = useTranslation('navigation');

  const navItems: NavItem[] = [
    { label: t('play'), icon: <IconPlayerPlay size={18} />, onClick: () => {} },
    { label: t('create'), icon: <IconEdit size={18} />, onClick: () => {} },
    { label: t('rooms'), icon: <IconBuilding size={18} />, onClick: () => {} },
    { label: t('groups'), icon: <IconUsers size={18} />, onClick: () => {} },
  ];

  return (
    <AppLayout 
      variant="authenticated" 
      navItems={navItems}
      headerProps={{
        onNotificationsClick: () => navigationLogger.debug('Notifications clicked'),
        onSettingsClick: () => navigationLogger.debug('Settings clicked'),
        onProfileClick: () => navigationLogger.debug('Profile clicked'),
      }}
    >
      <DashboardContent />
    </AppLayout>
  );
}
