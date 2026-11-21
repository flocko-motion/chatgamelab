import { createFileRoute, useSearch } from '@tanstack/react-router';
import { AllGames } from '@/features/games';
import { validateListSearch, type ListSearch } from '@/common/lib/listQuery';

type GamesSearch = ListSearch & {
  action?: 'import';
};

export const Route = createFileRoute('/games/')({
  component: AllGamesPage,
  validateSearch: (search: Record<string, unknown>): GamesSearch => ({
    ...validateListSearch(search),
    action: search.action === 'import' ? 'import' : undefined,
  }),
});

function AllGamesPage() {
  const { action } = useSearch({ from: '/games/' });
  return <AllGames autoImport={action === 'import'} />;
}
