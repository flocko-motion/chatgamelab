import { createFileRoute, redirect } from '@tanstack/react-router';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute('/creations/')({
  beforeLoad: () => {
    throw redirect({ to: ROUTES.ALL_GAMES, search: { only: 'own' } });
  },
  component: () => null,
});
