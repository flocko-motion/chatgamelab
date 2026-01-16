import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/creations/$gameId')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: `/my-games/${params.gameId}` as '/' });
  },
  component: () => null,
});
