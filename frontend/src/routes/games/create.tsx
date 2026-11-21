import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { AllGames } from '@/features/games';

export const Route = createFileRoute('/games/create')({
  component: CreateGamePage,
});

function CreateGamePage() {
  const navigate = useNavigate();
  return <AllGames initialCreate onModalClose={() => navigate({ to: '/games' })} />;
}
