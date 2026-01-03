import { createFileRoute, useRouter } from '@tanstack/react-router';
import { Container, Title, Text, Stack, Card, Group, Button, Badge } from '@mantine/core';
import { useAuth } from '@/providers/AuthProvider';

export const Route = createFileRoute('/dashboard')({
  component: Dashboard,
});

function Dashboard() {
  const { user, logout, isDevMode } = useAuth();
  const router = useRouter();

  const handleLogout = () => {
    logout();
    router.navigate({ to: '/' });
  };

  if (!user) {
    // Redirect to login if not authenticated
    router.navigate({ to: '/auth/login' });
    return null;
  }

  return (
    <Container size="lg" py="xl">
      <Stack gap="lg">
        {/* Header */}
        <Group justify="space-between">
          <div>
            <Title order={1}>Dashboard</Title>
            <Text size="lg" c="dimmed">
              Welcome back, {user.name || user.email || 'User'}!
            </Text>
          </div>
          <Button onClick={handleLogout} variant="outline" color="red">
            Logout
          </Button>
        </Group>

        {/* Dev Mode Indicator */}
        {isDevMode && (
          <Card withBorder p="md" bg="orange.0">
            <Group gap="sm">
              <Badge color="orange" variant="light">Development Mode</Badge>
              <Text size="sm">
                You are currently in development mode with role: {user.sub || 'dev-user'}
              </Text>
            </Group>
          </Card>
        )}

        {/* Main Content */}
        <Stack gap="md">
          <Title order={2}>Your Applications</Title>
          
          {/* Dummy App Cards */}
          <Card withBorder p="lg" shadow="sm">
            <Stack gap="md">
              <Title order={3}>🎮 Game Creator</Title>
              <Text c="dimmed">
                Create and manage text adventure games with AI-powered storytelling.
              </Text>
              <Group>
                <Badge color="green">Active</Badge>
                <Text size="sm" c="dimmed">Last used: 2 hours ago</Text>
              </Group>
              <Button variant="filled">Open App</Button>
            </Stack>
          </Card>

          <Card withBorder p="lg" shadow="sm">
            <Stack gap="md">
              <Title order={3}>📚 Game Library</Title>
              <Text c="dimmed">
                Browse and play games created by the community.
              </Text>
              <Group>
                <Badge color="blue">Available</Badge>
                <Text size="sm" c="dimmed">127 games available</Text>
              </Group>
              <Button variant="outline">Browse Library</Button>
            </Stack>
          </Card>

          <Card withBorder p="lg" shadow="sm" opacity={0.6}>
            <Stack gap="md">
              <Title order={3}>🎨 Theme Designer</Title>
              <Text c="dimmed">
                Customize the appearance of your games with themes and styles.
              </Text>
              <Group>
                <Badge color="gray">Coming Soon</Badge>
                <Text size="sm" c="dimmed">Under development</Text>
              </Group>
              <Button variant="light" disabled>
                Coming Soon
              </Button>
            </Stack>
          </Card>
        </Stack>

        {/* User Info Section */}
        <Card withBorder p="md" bg="gray.0">
          <Stack gap="sm">
            <Title order={4}>Account Information</Title>
            <Text size="sm">
              <strong>User ID:</strong> {user.sub}
            </Text>
            {user.email && (
              <Text size="sm">
                <strong>Email:</strong> {user.email}
              </Text>
            )}
            {user.name && (
              <Text size="sm">
                <strong>Name:</strong> {user.name}
              </Text>
            )}
            <Text size="sm" c="dimmed">
              <strong>Mode:</strong> {isDevMode ? 'Development' : 'Production'}
            </Text>
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
}
