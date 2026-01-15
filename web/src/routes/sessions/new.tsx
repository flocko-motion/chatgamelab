import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { GameSelection } from '@/features/play';
import { createGamePlayRoute } from '@/common/routes/routes';
import type { ObjGame } from '@/api/generated';

export const Route = createFileRoute('/sessions/new')({
  component: NewSessionPage,
});

function NewSessionPage() {
  const navigate = useNavigate();

  const handleSelectGame = (game: ObjGame) => {
    if (game.id) {
      navigate({ to: createGamePlayRoute(game.id) as '/' });
    }
  };

  return <GameSelection onSelectGame={handleSelectGame} />;
}
