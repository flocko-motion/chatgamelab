import { createFileRoute, redirect } from '@tanstack/react-router';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute('/creations/create')({
  beforeLoad: () => {
    throw redirect({ to: ROUTES.MY_GAMES + '/create' as '/' });
  },
  component: () => null,
});
