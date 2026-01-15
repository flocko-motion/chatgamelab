import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { GameSelection } from '@/features/play';
import { createGamePlayRoute } from '@/common/routes/routes';
import type { ObjGame } from '@/api/generated';

export const Route = createFileRoute('/play/')({
  component: PlayPage,
});

function PlayPage() {
  const navigate = useNavigate();

  const handleSelectGame = (game: ObjGame) => {
    if (game.id) {
      navigate({ to: createGamePlayRoute(game.id) as '/' });
    }
  };

  return <GameSelection onSelectGame={handleSelectGame} />;
}
