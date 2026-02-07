import { createFileRoute, useSearch } from '@tanstack/react-router';
import { MyGames } from '@/features/games';

type MyGamesSearch = {
  action?: 'import';
};

export const Route = createFileRoute('/my-games/')({
  component: MyGamesPage,
  validateSearch: (search: Record<string, unknown>): MyGamesSearch => ({
    action: search.action === 'import' ? 'import' : undefined,
  }),
});

function MyGamesPage() {
  const { action } = useSearch({ from: '/my-games/' });
  return <MyGames autoImport={action === 'import'} />;
}
