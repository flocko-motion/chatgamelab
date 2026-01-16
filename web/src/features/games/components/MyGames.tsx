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
  Badge,
} from '@mantine/core';
import { useDisclosure, useMediaQuery } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { IconPlus, IconAlertCircle, IconMoodEmpty, IconUpload, IconWorld, IconLock, IconCopy } from '@tabler/icons-react';
import { TextButton, PlayGameButton, EditIconButton, DeleteIconButton, GenericIconButton } from '@components/buttons';
import { SortSelector, type SortOption } from '@components/controls';
import { PageTitle } from '@components/typography';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { useGames, useCreateGame, useDeleteGame, useExportGameYaml, useImportGameYaml, useGameSessionMap, useDeleteSession, useCloneGame } from '@/api/hooks';
import type { ObjGame, DbUserSessionWithGame } from '@/api/generated';
import { type SortField, type CreateGameFormData } from '../types';
import { CreateGameModal } from './CreateGameModal';
import { DeleteGameModal } from './DeleteGameModal';
import { GameViewModal } from './GameViewModal';
import { IconDownload } from '@tabler/icons-react';
import { useModals } from '@mantine/modals';

interface MyGamesProps {
  initialGameId?: string;
  initialMode?: 'create' | 'view';
  onModalClose?: () => void;
}

export function MyGames({ initialGameId, initialMode, onModalClose }: MyGamesProps = {}) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');
  const navigate = useNavigate();
  const modals = useModals();
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(initialMode === 'create');
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] = useDisclosure(initialGameId ? true : false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [gameToView, setGameToView] = useState<string | null>(initialGameId ?? null);
  const [editMode, setEditMode] = useState(initialGameId ? true : false);
  const [sortField, setSortField] = useState<SortField>('modifiedAt');

  const { data: games, isLoading, error, refetch } = useGames({ sortBy: sortField, sortDir: 'desc', filter: 'own' });
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const createGame = useCreateGame();
  const deleteGame = useDeleteGame();
  const deleteSession = useDeleteSession();
  const exportGameYaml = useExportGameYaml();
  const importGameYaml = useImportGameYaml();
  const cloneGame = useCloneGame();
  
  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const handleCreateGame = async (data: CreateGameFormData) => {
    try {
      const newGame = await createGame.mutateAsync({
        name: data.name,
        description: data.description,
        public: data.isPublic,
      });
      closeCreateModal();
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

  const handleCopyGame = (game: ObjGame) => {
    if (!game.id) return;
    
    modals.openConfirmModal({
      title: t('myGames.copyConfirm.title'),
      children: (
        <Text size="sm">
          {t('myGames.copyConfirm.message', { name: game.name || t('sessions.untitledGame') })}
        </Text>
      ),
      labels: {
        confirm: t('myGames.copyConfirm.confirm'),
        cancel: t('cancel'),
      },
      onConfirm: async () => {
        const newGame = await cloneGame.mutateAsync(game.id!);
        if (newGame.id) {
          setGameToView(newGame.id);
          setEditMode(true);
          openViewModal();
        }
      },
    });
  };

  const handlePlayGame = (game: ObjGame) => {
    if (game.id) {
      navigate({ to: '/games/$gameId/play', params: { gameId: game.id } });
    }
  };

  const handleContinueGame = (session: DbUserSessionWithGame) => {
    if (session.id) {
      navigate({ to: `/sessions/${session.id}` as '/' });
    }
  };

  const handleRestartGame = (game: ObjGame, session: DbUserSessionWithGame) => {
    if (!game.id || !session.id) return;
    
    modals.openConfirmModal({
      title: t('myGames.restartConfirm.title'),
      children: (
        <Text size="sm">
          {t('myGames.restartConfirm.message', { game: game.name || t('sessions.untitledGame') })}
        </Text>
      ),
      labels: {
        confirm: t('myGames.restartConfirm.confirm'),
        cancel: t('cancel'),
      },
      confirmProps: { color: 'red' },
      onConfirm: async () => {
        await deleteSession.mutateAsync(session.id!);
        navigate({ to: '/games/$gameId/play', params: { gameId: game.id! } });
      },
    });
  };

  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      let newGameId: string | undefined;
      
      try {
        const nameMatch = content.match(/^name:\s*["']?(.+?)["']?\s*$/m);
        const gameName = nameMatch?.[1]?.trim() || file.name.replace(/\.(yaml|yml)$/i, '');
        
        const newGame = await createGame.mutateAsync({ name: gameName });
        newGameId = newGame.id;
        
        if (newGameId) {
          await importGameYaml.mutateAsync({ id: newGameId, yaml: content });
          refetch();
          setGameToView(newGameId);
          setEditMode(true);
          openViewModal();
        }
      } catch {
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

  const getGameSessionState = (game: ObjGame) => {
    if (!game.id) return { hasSession: false, session: undefined };
    const session = sessionMap.get(game.id);
    return { hasSession: !!session, session };
  };

  const renderPlayButton = (game: ObjGame) => {
    const { hasSession, session } = getGameSessionState(game);
    
    if (!hasSession) {
      return (
        <PlayGameButton onClick={() => handlePlayGame(game)} size="xs" style={{ width: '100%' }}>
          {t('myGames.play')}
        </PlayGameButton>
      );
    }
    
    return (
      <Stack gap={4}>
        <PlayGameButton onClick={() => handleContinueGame(session!)} size="xs" style={{ width: '100%' }}>
          {t('myGames.continue')}
        </PlayGameButton>
        <TextButton onClick={() => handleRestartGame(game, session!)} size="xs">
          {t('myGames.restart')}
        </TextButton>
      </Stack>
    );
  };

  const columns: DataTableColumn<ObjGame>[] = [
    {
      key: 'play',
      header: '',
      width: 140,
      render: (game) => (
        <div onClick={(e) => e.stopPropagation()}>
          {renderPlayButton(game)}
        </div>
      ),
    },
    {
      key: 'name',
      header: t('games.fields.name'),
      render: (game) => (
        <Stack gap={2}>
          <Group gap="xs" wrap="nowrap">
            <Text fw={600} size="sm" c="gray.8" lineClamp={1}>
              {game.name}
            </Text>
          </Group>
          {game.description && (
            <Text size="xs" c="gray.5" lineClamp={1}>
              {game.description}
            </Text>
          )}
        </Stack>
      ),
    },
    {
      key: 'visibility',
      header: t('games.fields.visibility'),
      width: 120,
      render: (game) =>
        game.public ? (
          <Badge size="sm" color="green" variant="light" leftSection={<IconWorld size={12} />}>
            {t('games.visibility.public')}
          </Badge>
        ) : (
          <Badge size="sm" color="gray" variant="light" leftSection={<IconLock size={12} />}>
            {t('games.visibility.private')}
          </Badge>
        ),
    },
    {
      key: 'actions',
      header: t('actions'),
      width: 120,
      render: (game) => (
        <Group gap="xs" onClick={(e) => e.stopPropagation()}>
          <EditIconButton onClick={() => handleEditGame(game)} aria-label={t('edit')} />
          <GenericIconButton
            icon={<IconCopy size={16} />}
            onClick={() => handleCopyGame(game)}
            aria-label={t('myGames.copyGame')}
          />
          <GenericIconButton
            icon={<IconDownload size={16} />}
            onClick={() => handleExport(game)}
            aria-label={t('games.importExport.exportButton')}
          />
          <DeleteIconButton onClick={() => handleDeleteClick(game)} aria-label={t('delete')} />
        </Group>
      ),
    },
  ];

  const sortOptions: SortOption[] = [
    { value: 'modifiedAt', label: t('games.sort.modifiedAt') },
    { value: 'createdAt', label: t('games.sort.createdAt') },
    { value: 'name', label: t('games.sort.name') },
  ];

  if (isLoading || sessionsLoading) {
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

  return (
    <Container size="lg" py="xl" h="calc(100vh - 210px)">
      <Stack gap="lg" h="100%">
        <PageTitle>{t('myGames.title')}</PageTitle>

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
          {(games?.length ?? 0) > 0 && (
            <SortSelector 
              options={sortOptions} 
              value={sortField} 
              onChange={(v) => setSortField(v as SortField)}
              label={t('games.sort.label')}
            />
          )}
        </Group>

        {isMobile ? (
          (games?.length ?? 0) === 0 ? (
            <Card shadow="sm" p="xl" radius="md" withBorder>
              <Stack align="center" gap="md" py="xl">
                <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
                <Text c="gray.6" ta="center">
                  {t('myGames.empty.title')}
                </Text>
                <Text size="sm" c="gray.5" ta="center">
                  {t('myGames.empty.description')}
                </Text>
              </Stack>
            </Card>
          ) : (
            <SimpleGrid cols={1} spacing="md">
              {games?.map((game) => (
                  <Card key={game.id} shadow="sm" p="lg" radius="md" withBorder onClick={() => handleViewGame(game)}>
                    <Stack gap="sm">
                      <Group gap="md" align="flex-start" wrap="nowrap">
                        <div onClick={(e) => e.stopPropagation()}>
                          {renderPlayButton(game)}
                        </div>
                        <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                          <Group gap="xs" wrap="nowrap">
                            <Text fw={600} size="md" lineClamp={1} style={{ flex: 1 }}>
                              {game.name}
                            </Text>
                            {game.public ? (
                              <Badge size="sm" color="green" variant="light" leftSection={<IconWorld size={12} />}>
                                {t('games.visibility.public')}
                              </Badge>
                            ) : (
                              <Badge size="sm" color="gray" variant="light" leftSection={<IconLock size={12} />}>
                                {t('games.visibility.private')}
                              </Badge>
                            )}
                          </Group>
                          {game.description && (
                            <Text size="sm" c="gray.6" lineClamp={2}>
                              {game.description}
                            </Text>
                          )}
                        </Stack>
                      </Group>
                      <Group justify="flex-end" gap="xs" onClick={(e) => e.stopPropagation()}>
                        <EditIconButton onClick={() => handleEditGame(game)} aria-label={t('edit')} />
                        <GenericIconButton
                          icon={<IconCopy size={16} />}
                          onClick={() => handleCopyGame(game)}
                          aria-label={t('myGames.copyGame')}
                        />
                        <GenericIconButton
                          icon={<IconDownload size={16} />}
                          onClick={() => handleExport(game)}
                          aria-label={t('games.importExport.exportButton')}
                        />
                        <DeleteIconButton onClick={() => handleDeleteClick(game)} aria-label={t('delete')} />
                      </Group>
                    </Stack>
                  </Card>
              ))}
            </SimpleGrid>
          )
        ) : (games?.length ?? 0) === 0 ? (
          <Card shadow="sm" p="xl" radius="md" withBorder>
            <Stack align="center" gap="md" py="xl">
              <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
              <Text c="gray.6" ta="center">
                {t('myGames.empty.title')}
              </Text>
              <Text size="sm" c="gray.5" ta="center">
                {t('myGames.empty.description')}
              </Text>
            </Stack>
          </Card>
        ) : (
          <DataTable
            data={games ?? []}
            columns={columns}
            getRowKey={(game) => game.id || ''}
            onRowClick={handleViewGame}
            isLoading={false}
            fillHeight
            renderMobileCard={(game) => (
              <Card shadow="sm" p="lg" radius="md" withBorder onClick={() => handleViewGame(game)}>
                <Stack gap="sm">
                  <Group gap="md" align="flex-start" wrap="nowrap">
                    <div onClick={(e) => e.stopPropagation()}>
                      {renderPlayButton(game)}
                    </div>
                    <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                      <Group gap="xs" wrap="nowrap">
                        <Text fw={600} size="md" lineClamp={1} style={{ flex: 1 }}>
                          {game.name}
                        </Text>
                        {game.public ? (
                          <Badge size="sm" color="green" variant="light" leftSection={<IconWorld size={12} />}>
                            {t('games.visibility.public')}
                          </Badge>
                        ) : (
                          <Badge size="sm" color="gray" variant="light" leftSection={<IconLock size={12} />}>
                            {t('games.visibility.private')}
                          </Badge>
                        )}
                      </Group>
                      {game.description && (
                        <Text size="sm" c="gray.6" lineClamp={2}>
                          {game.description}
                        </Text>
                      )}
                    </Stack>
                  </Group>
                  <Group justify="flex-end" gap="xs" onClick={(e) => e.stopPropagation()}>
                    <EditIconButton onClick={() => handleEditGame(game)} aria-label={t('edit')} />
                    <GenericIconButton
                      icon={<IconCopy size={16} />}
                      onClick={() => handleCopyGame(game)}
                      aria-label={t('myGames.copyGame')}
                    />
                    <GenericIconButton
                      icon={<IconDownload size={16} />}
                      onClick={() => handleExport(game)}
                      aria-label={t('games.importExport.exportButton')}
                    />
                    <DeleteIconButton onClick={() => handleDeleteClick(game)} aria-label={t('delete')} />
                  </Group>
                </Stack>
              </Card>
            )}
            emptyState={
              <DataTableEmptyState
                icon={<IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
                title={t('myGames.empty.title')}
                description={t('myGames.empty.description')}
              />
            }
          />
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
