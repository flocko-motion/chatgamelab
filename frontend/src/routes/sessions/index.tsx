import { createFileRoute, redirect } from '@tanstack/react-router';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute('/sessions/')({
  beforeLoad: () => {
    throw redirect({ to: ROUTES.ALL_GAMES });
  },
  component: () => null,
});
