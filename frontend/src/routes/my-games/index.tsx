import { createFileRoute, redirect } from '@tanstack/react-router';

// My Games is the games list with the "My games" toggle on.
export const Route = createFileRoute('/my-games/')({
  beforeLoad: ({ search }) => {
    const action = (search as { action?: unknown }).action === 'import' ? 'import' : undefined;
    throw redirect({ to: '/games', search: { only: 'own', action } });
  },
});
