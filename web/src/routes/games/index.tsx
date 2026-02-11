import { createFileRoute } from '@tanstack/react-router';
import { AllGames } from '@/features/games';

export const Route = createFileRoute('/games/')({
  component: AllGamesPage,
});

function AllGamesPage() {
  return <AllGames />;
}
