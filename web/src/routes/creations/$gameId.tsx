import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { GamesManagement } from '@/features/games';

export const Route = createFileRoute('/creations/$gameId')({
  component: CreationDetailPage,
});

function CreationDetailPage() {
  const { gameId } = Route.useParams();
  const navigate = useNavigate();

  // Pass gameId to GamesManagement to open modal directly
  // On modal close, navigate back to /creations
  return (
    <GamesManagement 
      initialGameId={gameId} 
      onModalClose={() => navigate({ to: '/creations' })}
    />
  );
}
