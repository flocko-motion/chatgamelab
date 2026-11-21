import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/my-games/create')({
  beforeLoad: () => {
    throw redirect({ to: '/games/create' });
  },
});
