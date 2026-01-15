import { createFileRoute } from '@tanstack/react-router';
import { Sessions } from '@/features/play';

export const Route = createFileRoute('/sessions/')({
  component: SessionsPage,
});

function SessionsPage() {
  return <Sessions />;
}
