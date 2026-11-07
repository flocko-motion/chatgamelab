import { createFileRoute } from '@tanstack/react-router';
import { PlayerPlaceholder } from '@/features/play/PlayerPlaceholder';

export const Route = createFileRoute('/games/$gameId/play')({
  component: PlayerPlaceholder,
});
