import { createFileRoute } from '@tanstack/react-router';
import { MyGames } from '@/features/games';

export const Route = createFileRoute('/my-games/')({
  component: MyGamesPage,
});

function MyGamesPage() {
  return <MyGames />;
}
