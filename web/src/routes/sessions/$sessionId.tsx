import { createFileRoute } from '@tanstack/react-router';
import { GamePlayer } from '@/features/game-player-v2';

export const Route = createFileRoute('/sessions/$sessionId')({
  component: SessionDetailPage,
});

function SessionDetailPage() {
  const { sessionId } = Route.useParams();
  
  return <GamePlayer sessionId={sessionId} />;
}
