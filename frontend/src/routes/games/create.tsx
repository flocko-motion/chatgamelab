import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router';
import { AllGames } from '@/features/games';

export const Route = createFileRoute('/games/create')({
  component: CreateGamePage,
  // ?about= starts the new game with what it is about (the dashboard's prompt).
  validateSearch: (search: Record<string, unknown>): { about?: string } => ({
    about: typeof search.about === 'string' && search.about ? search.about : undefined,
  }),
});

function CreateGamePage() {
  const navigate = useNavigate();
  const { about } = useSearch({ from: '/games/create' });
  return <AllGames initialCreate initialAbout={about} onModalClose={() => navigate({ to: '/games' })} />;
}
