import { 
  useGames, 
  useCreateGame, 
  useCurrentUser,
  useGameSessions,
  useCreateGameSession
} from '../../../api/client';
import type { ObjGame, ObjGameSession } from '../../../api/generated';
import { Button, Card, Text, Title, Stack, Group, LoadingOverlay } from '@mantine/core';

export function GameList() {
  const { data: games, isLoading, error } = useGames();
  const createGameMutation = useCreateGame();

  if (isLoading) return <LoadingOverlay visible />;
  if (error) return <Text c="red">Error loading games: {error.message}</Text>;

  const handleCreateGame = () => {
    createGameMutation.mutate({
      name: 'New Adventure Game',
    });
  };

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={2}>Games</Title>
        <Button 
          onClick={handleCreateGame}
          loading={createGameMutation.isPending}
        >
          Create Game
        </Button>
      </Group>
      
      {games?.map((game: ObjGame) => (
        <Card key={game.id} shadow="sm" p="md" withBorder>
          <Title order={3}>{game.name}</Title>
          <Text size="sm" c="dimmed">{game.description}</Text>
          <GameSessions gameId={game.id!} />
        </Card>
      ))}
    </Stack>
  );
}

function GameSessions({ gameId }: { gameId: string }) {
  const { data: sessions, isLoading } = useGameSessions(gameId);
  const createSessionMutation = useCreateGameSession();

  const handleCreateSession = () => {
    createSessionMutation.mutate({
      gameId,
      request: {
        model: 'gpt-4',
      },
    });
  };

  return (
    <Stack mt="md">
      <Group>
        <Title order={4}>Sessions</Title>
        <Button 
          size="sm"
          onClick={handleCreateSession}
          loading={createSessionMutation.isPending}
        >
          Start Session
        </Button>
      </Group>
      
      {isLoading && <Text size="sm" c="dimmed">Loading sessions...</Text>}
      
      {sessions?.map((session: ObjGameSession) => (
        <Text key={session.id} size="sm">
          Session {session.id} - {session.aiModel}
        </Text>
      ))}
    </Stack>
  );
}

export function UserProfile() {
  const { data: user, isLoading, error } = useCurrentUser();

  if (isLoading) return <LoadingOverlay visible />;
  if (error) return <Text c="red">Error loading user: {error.message}</Text>;

  return (
    <Card shadow="sm" p="md" withBorder>
      <Title order={3}>User Profile</Title>
      <Text>Name: {user?.name}</Text>
      <Text>Email: {user?.email}</Text>
      <Text>Role: {user?.role?.role || 'No role'}</Text>
    </Card>
  );
}

// Example of how to use the hooks in a component
export function Dashboard() {
  return (
    <Stack>
      <UserProfile />
      <GameList />
    </Stack>
  );
}
