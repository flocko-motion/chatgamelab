import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/creations/$gameId')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/games/$gameId', params: { gameId: params.gameId } });
  },
  component: () => null,
});
