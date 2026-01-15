import { useState, useMemo } from 'react';
import {
  Container,
  Stack,
  Group,
  Card,
  Alert,
  SimpleGrid,
  Skeleton,
  Text,
} from '@mantine/core';
import { useDisclosure, useMediaQuery } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { IconPlus, IconAlertCircle, IconMoodEmpty } from '@tabler/icons-react';
import { ActionButton, TextButton } from '@components/buttons';
import { SortSelector, type SortOption } from '@components/controls';
import { PageTitle } from '@components/typography';
import { useGames, useCreateGame, useDeleteGame } from '@/api/hooks';
import type { ObjGame } from '@/api/generated';
import { sortGames, type SortField, type CreateGameFormData } from '../types';
import { GamesTable } from './GamesTable';
import { GameCard } from './GameCard';
import { CreateGameModal } from './CreateGameModal';
import { DeleteGameModal } from './DeleteGameModal';
import { GameViewModal } from './GameViewModal';

interface GamesManagementProps {
  initialGameId?: string;
  initialMode?: 'create' | 'view';
  onModalClose?: () => void;
}

export function GamesManagement({ initialGameId, initialMode, onModalClose }: GamesManagementProps = {}) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(initialMode === 'create');
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] = useDisclosure(initialGameId ? true : false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [gameToView, setGameToView] = useState<string | null>(initialGameId ?? null);
  const [startInEditMode, setStartInEditMode] = useState(initialGameId ? true : false);
  const [sortField, setSortField] = useState<SortField>('modifiedAt');

  const { data: games, isLoading, error } = useGames();
  const createGame = useCreateGame();
  const deleteGame = useDeleteGame();

  const sortedGames = useMemo(() => {
    if (!games) return [];
    return sortGames(games, { field: sortField, direction: 'desc' });
  }, [games, sortField]);

  const handleCreateGame = async (data: CreateGameFormData) => {
    try {
      const newGame = await createGame.mutateAsync({
        name: data.name,
        description: data.description,
        public: data.isPublic,
      });
      closeCreateModal();
      // Open the new game in edit mode
      if (newGame.id) {
        setGameToView(newGame.id);
        setStartInEditMode(true);
        openViewModal();
      }
    } catch {
      // Error handled by mutation
    }
  };

  const handleCloseCreateModal = () => {
    closeCreateModal();
    if (initialMode === 'create') {
      onModalClose?.();
    }
  };

  const handleEditGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      setStartInEditMode(true);
      openViewModal();
    }
  };

  const handleViewGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      setStartInEditMode(true);
      openViewModal();
    }
  };

  const handleDeleteClick = (game: ObjGame) => {
    setGameToDelete(game);
    openDeleteModal();
  };

  const handleConfirmDelete = async () => {
    if (!gameToDelete?.id) return;
    try {
      await deleteGame.mutateAsync(gameToDelete.id);
      closeDeleteModal();
      setGameToDelete(null);
    } catch {
      // Error handled by mutation
    }
  };

  if (isLoading) {
    return (
      <Container size="lg" py="xl">
        <Stack gap="xl">
          <Skeleton height={40} width="50%" />
          <Skeleton height={36} width={180} />
          {isMobile ? (
            <Stack gap="md">
              {[1, 2, 3].map((i) => (
                <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
                  <Stack gap="sm">
                    <Skeleton height={24} width="70%" />
                    <Skeleton height={16} width="90%" />
                    <Group gap="xl">
                      <Skeleton height={32} width={80} />
                      <Skeleton height={32} width={80} />
                    </Group>
                  </Stack>
                </Card>
              ))}
            </Stack>
          ) : (
            <Skeleton height={300} />
          )}
        </Stack>
      </Container>
    );
  }

  if (error) {
    return (
      <Container size="lg" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t('errors.titles.error')} color="red">
          {t('games.errors.loadFailed')}
        </Alert>
      </Container>
    );
  }

  const sortOptions: SortOption[] = [
    { value: 'modifiedAt', label: t('games.sort.modifiedAt') },
    { value: 'createdAt', label: t('games.sort.createdAt') },
    { value: 'name', label: t('games.sort.name') },
  ];

  return (
    <Container size="lg" py="xl">
      <Stack gap="lg">
        <PageTitle>{t('games.title')}</PageTitle>

        {sortedGames.length > 0 && (
          <Group justify="space-between">
            <TextButton
              leftSection={<IconPlus size={16} />}
              onClick={openCreateModal}
            >
              {t('games.createButton')}
            </TextButton>
            <SortSelector 
              options={sortOptions} 
              value={sortField} 
              onChange={(v) => setSortField(v as SortField)}
              label={t('games.sort.label')}
            />
          </Group>
        )}

        {sortedGames.length === 0 ? (
          <Card shadow="sm" p="xl" radius="md" withBorder>
            <Stack align="center" gap="md" py="xl">
              <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
              <Text c="gray.6" ta="center">
                {t('games.empty.title')}
              </Text>
              <Text size="sm" c="gray.5" ta="center">
                {t('games.empty.description')}
              </Text>
              <ActionButton
                leftSection={<IconPlus size={18} />}
                onClick={openCreateModal}
              >
                {t('games.createButton')}
              </ActionButton>
            </Stack>
          </Card>
        ) : isMobile ? (
          <SimpleGrid cols={1} spacing="md">
            {sortedGames.map((game) => (
              <GameCard
                key={game.id}
                game={game}
                onView={handleViewGame}
                onEdit={handleEditGame}
                onDelete={handleDeleteClick}
              />
            ))}
          </SimpleGrid>
        ) : (
          <Card shadow="sm" p={0} radius="md" withBorder>
            <GamesTable
              games={sortedGames}
              onView={handleViewGame}
              onEdit={handleEditGame}
              onDelete={handleDeleteClick}
            />
          </Card>
        )}
      </Stack>

      <CreateGameModal
        opened={createModalOpened}
        onClose={handleCloseCreateModal}
        onSubmit={handleCreateGame}
        loading={createGame.isPending}
      />

      <DeleteGameModal
        opened={deleteModalOpened}
        onClose={() => {
          closeDeleteModal();
          setGameToDelete(null);
        }}
        onConfirm={handleConfirmDelete}
        gameName={gameToDelete?.name ?? ''}
        loading={deleteGame.isPending}
      />

      <GameViewModal
        gameId={gameToView}
        opened={viewModalOpened}
        onClose={() => {
          closeViewModal();
          setGameToView(null);
          setStartInEditMode(false);
          onModalClose?.();
        }}
        allowEdit
        startInEditMode={startInEditMode}
      />
    </Container>
  );
}
