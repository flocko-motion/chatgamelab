import { 
  Button, 
  Card, 
  Text, 
  Title, 
  Stack, 
  Group, 
  Grid,
  Avatar,
  ActionIcon,
  Divider,
  SimpleGrid,
  Box,
  ThemeIcon,
  Image,
  AppShell,
  Container
} from '@mantine/core';
import { 
  IconPlayerPlay, 
  IconEdit, 
  IconBuilding, 
  IconUsers, 
  IconSettings, 
  IconBell,
  IconUser,
  IconClock,
  IconTrendingUp,
  IconPlus
} from '@tabler/icons-react';
import { LanguageSwitcher } from '@components/LanguageSwitcher';
import logo from '@/assets/logos/black/ChatGameLab-Logo-2025-Landscape-Black-Black-Text-Transparent.png';

// Navigation Component
function AppNavigation() {
  return (
    <Group gap="lg">
      <Button 
        variant="subtle" 
        size="md"
        c="white"
        leftSection={<IconPlayerPlay size={18} />}
        styles={{
          root: { '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' } }
        }}
      >
        Play
      </Button>
      <Button 
        variant="subtle" 
        size="md"
        c="white"
        leftSection={<IconEdit size={18} />}
        styles={{
          root: { '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' } }
        }}
      >
        Create
      </Button>
      <Button 
        variant="subtle" 
        size="md"
        c="white"
        leftSection={<IconBuilding size={18} />}
        styles={{
          root: { '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' } }
        }}
      >
        Rooms
      </Button>
      <Button 
        variant="subtle" 
        size="md"
        c="white"
        leftSection={<IconUsers size={18} />}
        styles={{
          root: { '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' } }
        }}
      >
        Groups
      </Button>
    </Group>
  );
}

// User Actions Component
function UserActions() {
  return (
    <Group gap="sm">
      <LanguageSwitcher size="sm" variant="compact" />
      <ActionIcon 
        variant="subtle" 
        size="lg"
        c="white"
        styles={{
          root: { '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' } }
        }}
      >
        <IconBell size={20} />
      </ActionIcon>
      <ActionIcon 
        variant="subtle" 
        size="lg"
        c="white"
        styles={{
          root: { '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' } }
        }}
      >
        <IconSettings size={20} />
      </ActionIcon>
      <Avatar 
        size="md" 
        radius="xl" 
        style={{ 
          cursor: 'pointer',
          border: '2px solid rgba(255, 255, 255, 0.3)',
          backgroundColor: 'rgba(255, 255, 255, 0.1)',
          color: 'white'
        }}
      >
        <IconUser size={20} color="white" />
      </Avatar>
    </Group>
  );
}

// Dashboard Header with proper layout
function DashboardHeader() {
  return (
    <AppShell.Header 
      p="md" 
      style={{ 
        background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)',
        borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
        boxShadow: '0 2px 10px rgba(0, 0, 0, 0.3)'
      }}
    >
      <Group justify="space-between" align="center" h="100%">
        <AppNavigation />
        
        {/* Centered Logo - Larger */}
        <Box style={{ position: 'absolute', left: '50%', transform: 'translateX(-50%)' }}>
          <Image 
            src={logo} 
            alt="ChatGameLab Logo" 
            h={60}
            w="auto"
            fit="contain"
          />
        </Box>
        
        <UserActions />
      </Group>
    </AppShell.Header>
  );
}

// Quick Stats Cards
function QuickStatsCards() {
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
              Active Adventures
            </Text>
            <Title order={1} c="#1f2937" mb="xs">3</Title>
            <Text size="sm" c="#6b7280">2 stories in progress</Text>
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
              Story Drafts
            </Text>
            <Title order={1} c="#1f2937" mb="xs">2</Title>
            <Text size="sm" c="#6b7280">Ready for players</Text>
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
              Story Rooms
            </Text>
            <Title order={1} c="#1f2937" mb="xs">5</Title>
            <Text size="sm" c="#6b7280">Available sessions</Text>
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
              Storytellers
            </Text>
            <Title order={1} c="#1f2937" mb="xs">12</Title>
            <Text size="sm" c="#6b7280">Online now</Text>
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
  const activities = [
    {
      icon: <IconPlayerPlay size={16} />,
      color: 'violet',
      title: 'Space Adventure',
      user: 'Sarah',
      action: 'continued playing',
      time: '2 min ago',
      description: 'Chapter 3: The Dark Forest'
    },
    {
      icon: <IconEdit size={16} />,
      color: 'blue',
      title: 'Dragon Quest',
      user: 'Mike',
      action: 'created new game',
      time: '15 min ago',
      description: 'Fantasy adventure with magic'
    },
    {
      icon: <IconBuilding size={16} />,
      color: 'green',
      title: 'Workshop 3B',
      user: 'Emma',
      action: 'invited 5 students',
      time: '1 hour ago',
      description: 'Creative writing session'
    }
  ];

  return (
    <Card p="lg" withBorder shadow="sm">
      <Group justify="space-between" mb="md">
        <Title order={3}>Recent Activity</Title>
        <Button variant="subtle" size="sm">View All</Button>
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
  return (
    <Card p="lg" withBorder shadow="sm">
      <Title order={3} mb="md">Quick Actions</Title>
      <Stack gap="sm">
        <Button 
          variant="light" 
          leftSection={<IconPlus size={16} />}
          fullWidth
          justify="start"
        >
          Create New Game
        </Button>
        <Button 
          variant="light" 
          leftSection={<IconBuilding size={16} />}
          fullWidth
          justify="start"
        >
          Create Room
        </Button>
        <Button 
          variant="light" 
          leftSection={<IconUsers size={16} />}
          fullWidth
          justify="start"
        >
          Invite Members
        </Button>
        <Button 
          variant="light" 
          leftSection={<IconTrendingUp size={16} />}
          fullWidth
          justify="start"
        >
          View Analytics
        </Button>
      </Stack>
    </Card>
  );
}

// Main Dashboard Component
export function Dashboard() {
  return (
    <AppShell
      header={{ height: 80 }}
      padding="xl"
      style={{ 
        background: '#ffffff',
        minHeight: '100vh'
      }}
    >
      <DashboardHeader />
      <Container size="xl" fluid>
        <Stack gap="xl" mt="xl">
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
      </Container>
    </AppShell>
  );
}
