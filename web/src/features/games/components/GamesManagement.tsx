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
import { IconPlus, IconAlertCircle, IconMoodEmpty, IconUpload } from '@tabler/icons-react';
import { ActionButton, TextButton } from '@components/buttons';
import { SortSelector, type SortOption } from '@components/controls';
import { PageTitle } from '@components/typography';
import { useGames, useCreateGame, useDeleteGame, useExportGameYaml, useImportGameYaml } from '@/api/hooks';
import type { ObjGame } from '@/api/generated';
import { type SortField, type CreateGameFormData } from '../types';
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
  const [editMode, setEditMode] = useState(initialGameId ? true : false);
  const [sortField, setSortField] = useState<SortField>('modifiedAt');

  const { data: games, isLoading, error, refetch } = useGames({ sortBy: sortField, sortDir: 'desc' });
  const createGame = useCreateGame();
  const deleteGame = useDeleteGame();
  const exportGameYaml = useExportGameYaml();
  const importGameYaml = useImportGameYaml();
  
  // Import file input ref
  const fileInputRef = React.useRef<HTMLInputElement>(null);

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
        setEditMode(true);
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
      setEditMode(true);
      openViewModal();
    }
  };

  const handleViewGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      setEditMode(true);
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

  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      let newGameId: string | undefined;
      
      try {
        // Parse YAML to extract the name
        const nameMatch = content.match(/^name:\s*["']?(.+?)["']?\s*$/m);
        const gameName = nameMatch?.[1]?.trim() || file.name.replace(/\.(yaml|yml)$/i, '');
        
        // Create a new game with the extracted name
        const newGame = await createGame.mutateAsync({ name: gameName });
        newGameId = newGame.id;
        
        if (newGameId) {
          // Update the game with the full YAML content
          await importGameYaml.mutateAsync({ id: newGameId, yaml: content });
          refetch();
          // Open the imported game in edit mode
          setGameToView(newGameId);
          setEditMode(true);
          openViewModal();
        }
      } catch {
        // If import failed and we created a game, delete it
        if (newGameId) {
          try {
            await deleteGame.mutateAsync(newGameId);
            refetch();
          } catch {
            // Ignore cleanup errors
          }
        }
      }
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

        {(games?.length ?? 0) > 0 && (
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
        )}

        {(games?.length ?? 0) === 0 ? (
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
            {games?.map((game) => (
              <GameCard
                key={game.id}
                game={game}
                onView={handleViewGame}
                onEdit={handleEditGame}
                onDelete={handleDeleteClick}
                onExport={handleExport}
              />
            ))}
          </SimpleGrid>
        ) : (
          <Card shadow="sm" p={0} radius="md" withBorder style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
            <GamesTable
              games={games ?? []}
              onView={handleViewGame}
              onEdit={handleEditGame}
              onDelete={handleDeleteClick}
              onExport={handleExport}
              fillHeight
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
          setEditMode(false);
          onModalClose?.();
        }}
        editMode={editMode}
      />
    </Container>
  );
}
