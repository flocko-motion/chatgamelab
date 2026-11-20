import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { MyGames } from '@/features/games';
import { useAdmin } from '@/common/hooks/useAdmin';
import { ROUTES } from '@/common/routes/routes';

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
  const { isAdmin } = useAdmin();
  const navigate = useNavigate();

  useEffect(() => {
    if (isAdmin) {
      navigate({ to: ROUTES.ALL_GAMES as '/' });
    }
  }, [isAdmin, navigate]);

  if (isAdmin) return null;

  return <MyGames autoImport={action === 'import'} />;
}
