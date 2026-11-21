import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/my-games/$gameId')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/games/$gameId', params: { gameId: params.gameId } });
  },
});
