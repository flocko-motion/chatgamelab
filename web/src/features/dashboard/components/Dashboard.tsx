import { 
  Button, 
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
import { useTranslation } from 'react-i18next';
import { 
  IconPlayerPlay, 
  IconEdit, 
  IconBuilding, 
  IconUsers, 
  IconClock,
  IconTrendingUp,
  IconPlus
} from '@tabler/icons-react';
import { AppLayout, type NavItem } from '@/common/components/Layout';

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
          background: '#ffffff',
          borderColor: '#e1e5e9',
          borderTop: '3px solid #8b5cf6'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Text size="sm" tt="uppercase" fw={600} c="#6b7280" mb="xs">
              {t('stats.activeAdventures')}
            </Text>
            <Title order={1} c="#1f2937" mb="xs">3</Title>
            <Text size="sm" c="#6b7280">{t('stats.storiesInProgress', { count: 2 })}</Text>
          </div>
          <ThemeIcon 
            size={48} 
            radius="md"
            variant="light"
            color="violet"
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
          background: '#ffffff',
          borderColor: '#e1e5e9',
          borderTop: '3px solid #3b82f6'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Text size="sm" tt="uppercase" fw={600} c="#6b7280" mb="xs">
              {t('stats.storyDrafts')}
            </Text>
            <Title order={1} c="#1f2937" mb="xs">2</Title>
            <Text size="sm" c="#6b7280">{t('stats.readyForPlayers')}</Text>
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
          background: '#ffffff',
          borderColor: '#e1e5e9',
          borderTop: '3px solid #10b981'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Text size="sm" tt="uppercase" fw={600} c="#6b7280" mb="xs">
              {t('stats.storyRooms')}
            </Text>
            <Title order={1} c="#1f2937" mb="xs">5</Title>
            <Text size="sm" c="#6b7280">{t('stats.availableSessions')}</Text>
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
          background: '#ffffff',
          borderColor: '#e1e5e9',
          borderTop: '3px solid #f59e0b'
        }}
      >
        <Group justify="space-between" align="flex-start">
          <div>
            <Text size="sm" tt="uppercase" fw={600} c="#6b7280" mb="xs">
              {t('stats.storytellers')}
            </Text>
            <Title order={1} c="#1f2937" mb="xs">12</Title>
            <Text size="sm" c="#6b7280">{t('stats.onlineNow')}</Text>
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
      color: 'violet',
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
        <Title order={3}>{t('recentActivity.title')}</Title>
        <Button variant="subtle" size="sm">{t('recentActivity.viewAll')}</Button>
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
      <Title order={3} mb="md">{t('quickActions.title')}</Title>
      <Stack gap="sm">
        <Button 
          variant="light" 
          leftSection={<IconPlus size={16} />}
          fullWidth
          justify="start"
        >
          {t('quickActions.createNewGame')}
        </Button>
        <Button 
          variant="light" 
          leftSection={<IconBuilding size={16} />}
          fullWidth
          justify="start"
        >
          {t('quickActions.createRoom')}
        </Button>
        <Button 
          variant="light" 
          leftSection={<IconUsers size={16} />}
          fullWidth
          justify="start"
        >
          {t('quickActions.inviteMembers')}
        </Button>
        <Button 
          variant="light" 
          leftSection={<IconTrendingUp size={16} />}
          fullWidth
          justify="start"
        >
          {t('quickActions.viewAnalytics')}
        </Button>
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
        onNotificationsClick: () => console.log('Notifications clicked'),
        onSettingsClick: () => console.log('Settings clicked'),
        onProfileClick: () => console.log('Profile clicked'),
      }}
    >
      <DashboardContent />
    </AppLayout>
  );
}
