import { createFileRoute } from '@tanstack/react-router';
import { MyGames } from '@/features/games';

export const Route = createFileRoute('/my-games/$gameId')({
  component: MyGameDetailPage,
});

function MyGameDetailPage() {
  const { gameId } = Route.useParams();
  return <MyGames initialGameId={gameId} initialMode="view" />;
}
