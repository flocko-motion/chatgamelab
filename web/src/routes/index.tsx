import { createFileRoute } from '@tanstack/react-router';
import { Title, Text, Stack, Card, Group, Badge } from '@mantine/core';

export const Route = createFileRoute('/')({
  component: HomePage,
});

function HomePage() {
  return (
    <Stack gap="lg">
      <Title order={1}>Welcome to ChatGameLab</Title>
      <Text c="dimmed" size="lg">
        Create your own GPT-powered text adventure games and play them with your friends.
      </Text>

      <Group>
        <Badge color="violet" size="lg">Educational</Badge>
        <Badge color="blue" size="lg">Interactive</Badge>
        <Badge color="green" size="lg">AI-Powered</Badge>
      </Group>

      <Card shadow="sm" padding="lg" radius="md" withBorder>
        <Text fw={500} size="lg" mb="xs">Getting Started</Text>
        <Text c="dimmed">
          This is a placeholder home page. The project structure is now set up with
          TanStack Router, TanStack Query, and Mantine UI.
        </Text>
      </Card>
    </Stack>
  );
}
