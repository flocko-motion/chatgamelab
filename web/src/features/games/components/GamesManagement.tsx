import React, { useState } from 'react';
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
import { useNavigate } from '@tanstack/react-router';
import { IconPlus, IconAlertCircle, IconMoodEmpty, IconUpload, IconEye, IconEdit, IconTrash, IconDownload } from '@tabler/icons-react';
import { TextButton } from '@components/buttons';
import { SortSelector, type SortOption } from '@components/controls';
import { PageTitle } from '@components/typography';
import { useGames, useCreateGame, useUpdateGame, useDeleteGame, useExportGameYaml } from '@/api/hooks';
import type { ObjGame } from '@/api/generated';
import { type SortField, type CreateGameFormData } from '../types';
import { parseGameYaml } from '../lib';
import { GamesTable } from './GamesTable';
import { GameCard } from './GameCard';
import { GameEditModal } from './GameEditModal';
import { DeleteGameModal } from './DeleteGameModal';

interface GamesManagementProps {
  initialGameId?: string;
  initialMode?: 'create' | 'view';
  onModalClose?: () => void;
}

export function GamesManagement({ initialGameId, initialMode, onModalClose }: GamesManagementProps = {}) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');
  const navigate = useNavigate();
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(initialMode === 'create');
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] = useDisclosure(initialGameId ? true : false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [gameToView, setGameToView] = useState<string | null>(initialGameId ?? null);
  const [sortField, setSortField] = useState<SortField>('modifiedAt');

  const { data: games, isLoading, error } = useGames({ sortBy: sortField, sortDir: 'desc' });
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const deleteGame = useDeleteGame();
  const exportGameYaml = useExportGameYaml();
  
  // Import file input ref
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  // Pre-populated data for create modal (from YAML import)
  const [createInitialData, setCreateInitialData] = useState<Partial<CreateGameFormData> | null>(null);

  const handleCreateGame = async (data: CreateGameFormData) => {
    try {
      const newGame = await createGame.mutateAsync({
        name: data.name,
        description: data.description,
        public: data.isPublic,
      });
      
      // Update with additional fields if provided
      const hasExtraFields = data.systemMessageScenario || data.systemMessageGameStart || data.imageStyle || data.statusFields;
      if (newGame.id && hasExtraFields) {
        await updateGame.mutateAsync({
          id: newGame.id,
          game: {
            ...newGame,
            systemMessageScenario: data.systemMessageScenario,
            systemMessageGameStart: data.systemMessageGameStart,
            imageStyle: data.imageStyle,
            statusFields: data.statusFields,
          },
        });
      }
      
      closeCreateModal();
    } catch {
      // Error handled by mutation
    }
  };

  const handleCloseCreateModal = () => {
    closeCreateModal();
    setCreateInitialData(null);
    if (initialMode === 'create') {
      onModalClose?.();
    }
  };

  const handleEditGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      openViewModal();
    }
  };

  const handleViewGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
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

  const handleExport = async (game: ObjGame) => {
    if (!game.id) return;
    try {
      const yaml = await exportGameYaml.mutateAsync(game.id);
      const blob = new Blob([yaml], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${game.name || 'game'}.yaml`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch {
      // Error handled by mutation
    }
  };

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handlePlayGame = (game: ObjGame) => {
    if (game.id) {
      navigate({ to: '/games/$gameId/play', params: { gameId: game.id } });
    }
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      const formData = parseGameYaml(content);
      setCreateInitialData(formData);
      openCreateModal();
    };
    reader.readAsText(file);
    event.target.value = '';
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
    <Container size="lg" py="xl" h="calc(100vh - 210px)">
      <Stack gap="lg" h="100%">
        <PageTitle>{t('games.title')}</PageTitle>

        <input
          type="file"
          ref={fileInputRef}
          onChange={handleFileSelect}
          accept=".yaml,.yml"
          style={{ display: 'none' }}
        />

        <Group justify="space-between">
          <Group gap="sm">
            <TextButton
              leftSection={<IconPlus size={16} />}
              onClick={openCreateModal}
            >
              {t('games.createButton')}
            </TextButton>
            <TextButton
              leftSection={<IconUpload size={16} />}
              onClick={handleImportClick}
            >
              {t('games.importExport.importButton')}
            </TextButton>
          </Group>
          <SortSelector 
            options={sortOptions} 
            value={sortField} 
            onChange={(v) => setSortField(v as SortField)}
            label={t('games.sort.label')}
          />
        </Group>

        {isMobile ? (
          (games?.length ?? 0) === 0 ? (
            <Card shadow="sm" p="xl" radius="md" withBorder>
              <Stack align="center" gap="md" py="xl">
                <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
                <Text c="gray.6" ta="center">
                  {t('games.empty.title')}
                </Text>
                <Text size="sm" c="gray.5" ta="center">
                  {t('games.empty.description')}
                </Text>
              </Stack>
            </Card>
          ) : (
            <SimpleGrid cols={1} spacing="md">
              {games?.map((game) => (
                <GameCard
                  key={game.id}
                  game={game}
                  onPlay={() => handlePlayGame(game)}
                  playLabel={t('games.actions.play')}
                  showVisibility
                  actions={[
                    { key: 'view', icon: <IconEye size={16} />, label: t('games.actions.view'), onClick: () => handleViewGame(game) },
                    { key: 'edit', icon: <IconEdit size={16} />, label: t('games.actions.edit'), onClick: () => handleEditGame(game) },
                    { key: 'export', icon: <IconDownload size={16} />, label: t('games.actions.export'), onClick: () => handleExport(game) },
                    { key: 'delete', icon: <IconTrash size={16} />, label: t('games.actions.delete'), onClick: () => handleDeleteClick(game) },
                  ]}
                />
              ))}
            </SimpleGrid>
          )
        ) : (games?.length ?? 0) === 0 ? (
          <Card shadow="sm" p="xl" radius="md" withBorder>
            <Stack align="center" gap="md" py="xl">
              <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
              <Text c="gray.6" ta="center">
                {t('games.empty.title')}
              </Text>
              <Text size="sm" c="gray.5" ta="center">
                {t('games.empty.description')}
              </Text>
            </Stack>
          </Card>
        ) : (
          <Card shadow="sm" p={0} radius="md" withBorder style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
            <GamesTable
              games={games ?? []}
              onView={handleViewGame}
              onEdit={handleEditGame}
              onDelete={handleDeleteClick}
              onExport={handleExport}
              onPlay={handlePlayGame}
              fillHeight
            />
          </Card>
        )}
      </Stack>

      <GameEditModal
        opened={createModalOpened}
        onClose={handleCloseCreateModal}
        onCreate={handleCreateGame}
        createLoading={createGame.isPending}
        initialData={createInitialData}
      />

      <GameEditModal
        gameId={gameToView}
        opened={viewModalOpened}
        onClose={() => {
          closeViewModal();
          setGameToView(null);
          onModalClose?.();
        }}
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
    </Container>
  );
}
