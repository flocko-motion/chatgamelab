import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { AllGames } from '@/features/games';

export const Route = createFileRoute('/games/$gameId/')({
  component: GameDetailPage,
});

function GameDetailPage() {
  const { gameId } = Route.useParams();
  const navigate = useNavigate();
  return <AllGames initialGameId={gameId} onModalClose={() => navigate({ to: '/games' })} />;
}
