import { createFileRoute } from '@tanstack/react-router';
import { GamesManagement } from '@/features/games';

export const Route = createFileRoute('/creations/')({
  component: CreationsPage,
});

function CreationsPage() {
  return <GamesManagement />;
}
