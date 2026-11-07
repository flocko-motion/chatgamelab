import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { MyGames } from '@/features/games';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute('/my-games/create')({
  component: CreateGamePage,
});

function CreateGamePage() {
  const navigate = useNavigate();

  const handleClose = () => {
    navigate({ to: ROUTES.MY_GAMES as '/' });
  };

  return <MyGames initialMode="create" onModalClose={handleClose} />;
}
