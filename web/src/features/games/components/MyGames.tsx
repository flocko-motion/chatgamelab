import React, { useState } from 'react';
import {
  Stack,
  Group,
  Card,
  Alert,
  SimpleGrid,
  Skeleton,
  Text,
  Badge,
  Tooltip,
  Box,
} from '@mantine/core';
import { useDisclosure, useMediaQuery } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { IconPlus, IconAlertCircle, IconMoodEmpty, IconUpload, IconCopy, IconDownload, IconWorld, IconLock, IconStar, IconStarFilled } from '@tabler/icons-react';
import { TextButton, PlayGameButton, EditIconButton, DeleteIconButton, GenericIconButton } from '@components/buttons';
import { SortSelector, type SortOption, FilterSegmentedControl } from '@components/controls';
import { PageTitle } from '@components/typography';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { DimmedLoader } from '@components/LoadingAnimation';
import { useGames, useCreateGame, useDeleteGame, useExportGameYaml, useImportGameYaml, useGameSessionMap, useDeleteSession, useCloneGame, useFavoriteGames, useAddFavorite, useRemoveFavorite } from '@/api/hooks';
import type { ObjGame, DbUserSessionWithGame } from '@/api/generated';
import { type SortField, type CreateGameFormData } from '../types';
import { GameEditModal } from './GameEditModal';
import { DeleteGameModal } from './DeleteGameModal';
import { GameCard, type GameCardAction } from './GameCard';
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
  const [sortField, setSortField] = useState<SortField>('modifiedAt');
  const [showFavorites, setShowFavorites] = useState<'all' | 'favorites'>('all');

  const { data: rawGames, isLoading, isFetching, error, refetch } = useGames({ sortBy: sortField, sortDir: 'desc', filter: 'own' });
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const createGame = useCreateGame();
  const deleteGame = useDeleteGame();
  const deleteSession = useDeleteSession();
  const exportGameYaml = useExportGameYaml();
  const importGameYaml = useImportGameYaml();
  const cloneGame = useCloneGame();
  const { data: favoriteGames } = useFavoriteGames();
  const addFavorite = useAddFavorite();
  const removeFavorite = useRemoveFavorite();

  const favoriteGameIds = new Set(favoriteGames?.map(g => g.id) ?? []);
  
  // Apply client-side favorites filter
  const games = showFavorites === 'favorites' 
    ? rawGames?.filter(game => game.id && favoriteGameIds.has(game.id))
    : rawGames;

  const isFavorite = (game: ObjGame) => game.id ? favoriteGameIds.has(game.id) : false;

  const handleToggleFavorite = (game: ObjGame) => {
    if (!game.id) return;
    if (isFavorite(game)) {
      removeFavorite.mutate(game.id);
    } else {
      addFavorite.mutate(game.id);
    }
  };
  
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

  const getCardActions = (game: ObjGame): GameCardAction[] => [
    {
      key: 'edit',
      icon: null,
      label: t('editGame'),
      onClick: () => handleEditGame(game),
    },
    {
      key: 'copy',
      icon: <IconCopy size={16} />,
      label: t('copyGame'),
      onClick: () => handleCopyGame(game),
    },
    {
      key: 'export',
      icon: <IconDownload size={16} />,
      label: t('games.importExport.exportButton'),
      onClick: () => handleExport(game),
    },
    {
      key: 'delete',
      icon: null,
      label: t('deleteGame'),
      onClick: () => handleDeleteClick(game),
    },
  ];

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
      key: 'favorite',
      header: '',
      width: 40,
      render: (game) => (
        <div onClick={(e) => e.stopPropagation()}>
          <Tooltip label={isFavorite(game) ? t('myGames.unfavorite') : t('myGames.favorite')} withArrow>
            <GenericIconButton
              icon={isFavorite(game) ? <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" /> : <IconStar size={18} />}
              onClick={() => handleToggleFavorite(game)}
              aria-label={isFavorite(game) ? t('myGames.unfavorite') : t('myGames.favorite')}
            />
          </Tooltip>
        </div>
      ),
    },
    {
      key: 'play',
      header: '',
      width: 120,
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
      key: 'playCount',
      header: t('games.fields.playCount'),
      width: 80,
      render: (game) => (
        <Tooltip label={t('games.fields.playCount')} withArrow>
          <Text size="sm" c="gray.6" ta="center">
            {game.playCount ?? 0}
          </Text>
        </Tooltip>
      ),
    },
    {
      key: 'visibility',
      header: t('games.fields.visibility'),
      width: 150,
      render: (game) =>
        game.public ? (
          <Badge size="sm" color="green" variant="light" leftSection={<IconWorld size={12} />} style={{ whiteSpace: 'nowrap' }}>
            {t('games.visibility.public')}
          </Badge>
        ) : (
          <Badge size="sm" color="gray" variant="light" leftSection={<IconLock size={12} />} style={{ whiteSpace: 'nowrap' }}>
            {t('games.visibility.private')}
          </Badge>
        ),
    },
    {
      key: 'date',
      header: sortField === 'createdAt' ? t('games.fields.created') : t('games.fields.modified'),
      width: 100,
      render: (game) => {
        const dateValue = sortField === 'createdAt' ? game.meta?.createdAt : game.meta?.modifiedAt;
        const date = dateValue ? new Date(dateValue) : null;
        return (
          <Tooltip 
            label={date ? date.toLocaleString() : '-'} 
            withArrow 
            disabled={!date}
          >
            <Text size="sm" c="gray.6">
              {date ? date.toLocaleDateString() : '-'}
            </Text>
          </Tooltip>
        );
      },
    },
    {
      key: 'actions',
      header: t('actions'),
      width: 120,
      render: (game) => (
        <Group gap="xs" onClick={(e) => e.stopPropagation()}>
          <Tooltip label={t('editGame')} withArrow>
            <EditIconButton onClick={() => handleEditGame(game)} aria-label={t('edit')} />
          </Tooltip>
          <Tooltip label={t('copyGame')} withArrow>
            <GenericIconButton
              icon={<IconCopy size={16} />}
              onClick={() => handleCopyGame(game)}
              aria-label={t('myGames.copyGame')}
            />
          </Tooltip>
          <Tooltip label={t('games.importExport.exportButton')} withArrow>
            <GenericIconButton
              icon={<IconDownload size={16} />}
              onClick={() => handleExport(game)}
              aria-label={t('games.importExport.exportButton')}
            />
          </Tooltip>
          <Tooltip label={t('deleteGame')} withArrow>
            <DeleteIconButton onClick={() => handleDeleteClick(game)} aria-label={t('delete')} />
          </Tooltip>
        </Group>
      ),
    },
  ];

  const sortOptions: SortOption[] = [
    { value: 'modifiedAt', label: t('games.sort.modifiedAt') },
    { value: 'createdAt', label: t('games.sort.createdAt') },
    { value: 'name', label: t('games.sort.name') },
    { value: 'playCount', label: t('games.sort.playCount') },
    { value: 'visibility', label: t('games.sort.visibility') },
  ];

  const hasData = rawGames !== undefined;
  const isInitialLoading = !hasData && (isLoading || sessionsLoading);
  const isRefetching = isFetching && hasData;

  if (isInitialLoading) {
    return (
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
    );
  }

  if (error) {
    return (
      <Alert icon={<IconAlertCircle size={16} />} title={t('errors.titles.error')} color="red">
          {t('games.errors.loadFailed')}
      </Alert>
    );
  }

  return (
    <>
      <Stack gap="lg" h={{ base: 'calc(100vh - 180px)', sm: 'calc(100vh - 280px)' }} style={{ overflow: 'hidden' }}>
        {/* Sticky header section */}
        <Stack gap="lg" style={{ flexShrink: 0 }}>
          <PageTitle>{t('myGames.title')}</PageTitle>

          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileSelect}
            accept=".yaml,.yml"
            style={{ display: 'none' }}
          />

          <Group justify="space-between" wrap="wrap" gap="sm">
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
            <Group gap="sm" wrap="wrap">
              <FilterSegmentedControl
                value={showFavorites}
                onChange={setShowFavorites}
                options={[
                  { value: 'all', label: t('myGames.filters.all') },
                  { value: 'favorites', label: t('myGames.filters.favorites') },
                ]}
              />
              {(rawGames?.length ?? 0) > 0 && (
                <SortSelector 
                  options={sortOptions} 
                  value={sortField} 
                  onChange={(v) => setSortField(v as SortField)}
                  label={t('games.sort.label')}
                />
              )}
            </Group>
          </Group>
        </Stack>

        {/* Scrollable content area */}
        <Box style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
        <DimmedLoader visible={isRefetching} loaderSize="lg">
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
              {games?.map((game) => {
                const { hasSession, session } = getGameSessionState(game);
                return (
                  <GameCard
                    key={game.id}
                    game={game}
                    onClick={() => handleViewGame(game)}
                    onPlay={() => handlePlayGame(game)}
                    playLabel={t('myGames.play')}
                    hasSession={hasSession}
                    onContinue={session ? () => handleContinueGame(session) : undefined}
                    continueLabel={t('myGames.continue')}
                    onRestart={session ? () => handleRestartGame(game, session) : undefined}
                    restartLabel={t('myGames.restart')}
                    showVisibility
                    isFavorite={isFavorite(game)}
                    onToggleFavorite={() => handleToggleFavorite(game)}
                    favoriteLabel={t('myGames.favorite')}
                    unfavoriteLabel={t('myGames.unfavorite')}
                    actions={getCardActions(game)}
                  />
                );
              })}
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
            renderMobileCard={(game) => {
              const { hasSession, session } = getGameSessionState(game);
              return (
                <GameCard
                  game={game}
                  onClick={() => handleViewGame(game)}
                  onPlay={() => handlePlayGame(game)}
                  playLabel={t('myGames.play')}
                  hasSession={hasSession}
                  onContinue={session ? () => handleContinueGame(session) : undefined}
                  continueLabel={t('myGames.continue')}
                  onRestart={session ? () => handleRestartGame(game, session) : undefined}
                  restartLabel={t('myGames.restart')}
                  showVisibility
                  isFavorite={isFavorite(game)}
                  onToggleFavorite={() => handleToggleFavorite(game)}
                  favoriteLabel={t('myGames.favorite')}
                  unfavoriteLabel={t('myGames.unfavorite')}
                  actions={getCardActions(game)}
                />
              );
            }}
            emptyState={
              <DataTableEmptyState
                icon={<IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
                title={t('myGames.empty.title')}
                description={t('myGames.empty.description')}
              />
            }
          />
        )}
        </DimmedLoader>
        </Box>
      </Stack>

      <GameEditModal
        opened={createModalOpened}
        onClose={handleCloseCreateModal}
        onCreate={handleCreateGame}
        createLoading={createGame.isPending}
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
    </>
  );
}
