import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { GamesManagement } from '@/features/games';

export const Route = createFileRoute('/creations/create')({
  component: CreateGamePage,
});

function CreateGamePage() {
  const navigate = useNavigate();

  return (
    <GamesManagement 
      initialMode="create"
      onModalClose={() => navigate({ to: '/creations' })}
    />
  );
}
