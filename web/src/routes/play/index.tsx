import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';
import { PlayGamesList } from '@/features/play';
import { createGamePlayRoute } from '@/common/routes/routes';
import { useCloneGame } from '@/api/hooks';
import type { ObjGame } from '@/api/generated';

export const Route = createFileRoute('/play/')({
  component: PlayPage,
});

function PlayPage() {
  const navigate = useNavigate();
  const { t } = useTranslation('common');
  const cloneGame = useCloneGame();

  const handlePlay = (game: ObjGame) => {
    if (game.id) {
      navigate({ to: createGamePlayRoute(game.id) as '/' });
    }
  };

  const handleClone = async (game: ObjGame) => {
    if (!game.id) return;

    try {
      const clonedGame = await cloneGame.mutateAsync(game.id);
      
      notifications.show({
        title: t('play.cloneSuccess.title'),
        message: t('play.cloneSuccess.message', { name: clonedGame.name }),
        color: 'green',
      });

      // Navigate to edit the cloned game
      if (clonedGame.id) {
        navigate({ to: `/creations/${clonedGame.id}` as '/' });
      }
    } catch {
      // Error is handled by the mutation's onError
    }
  };

  return <PlayGamesList onPlay={handlePlay} onClone={handleClone} isCloning={cloneGame.isPending} />;
}
