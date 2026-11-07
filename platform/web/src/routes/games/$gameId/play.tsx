import { createFileRoute } from '@tanstack/react-router';
import { GamePlayer } from '@/features/game-player-v2';

export const Route = createFileRoute('/games/$gameId/play')({
  component: GamePlayPage,
});

function GamePlayPage() {
  const { gameId } = Route.useParams();
  
  return <GamePlayer gameId={gameId} />;
}
