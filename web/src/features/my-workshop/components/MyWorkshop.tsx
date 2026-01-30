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
import { useDisclosure, useMediaQuery, useDebouncedValue } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { IconAlertCircle, IconMoodEmpty, IconCopy, IconDownload, IconSchool, IconWorld, IconUser, IconEye } from '@tabler/icons-react';
import { ActionButton, PlayGameButton, EditIconButton, DeleteIconButton, GenericIconButton } from '@components/buttons';
import { SortSelector, type SortOption, FilterSegmentedControl, ExpandableSearch } from '@components/controls';
import { PageTitle } from '@components/typography';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { DimmedLoader } from '@components/LoadingAnimation';
import { useGames, useCreateGame, useUpdateGame, useDeleteGame, useExportGameYaml, useImportGameYaml, useGameSessionMap, useDeleteSession, useCloneGame } from '@/api/hooks';
import type { ObjGame, DbUserSessionWithGame } from '@/api/generated';
import { type CreateGameFormData } from '@/features/games/types';
import { GameEditModal, DeleteGameModal, GameCard, type GameCardAction } from '@/features/games';
import { useModals } from '@mantine/modals';
import { useAuth } from '@/providers/AuthProvider';
import { hasRole, Role } from '@/common/lib/roles';

type GameFilter = 'all' | 'mine' | 'workshop' | 'public';

export function MyWorkshop() {
  const { t } = useTranslation('common');
  const { t: tWorkshop } = useTranslation('myWorkshop');
  const isMobile = useMediaQuery('(max-width: 48em)');
  const navigate = useNavigate();
  const modals = useModals();
  const { backendUser } = useAuth();
  
  // Check if user can edit all workshop games (Head/Staff)
  const canEditAllWorkshopGames = hasRole(backendUser, Role.Head) || hasRole(backendUser, Role.Staff);
  const currentUserId = backendUser?.id;
  const workshopName = backendUser?.role?.workshop?.name;
  const organizationName = backendUser?.role?.institution?.name;
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] = useDisclosure(false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [gameToView, setGameToView] = useState<string | null>(null);
  const [gameToViewReadOnly, setGameToViewReadOnly] = useState(false);
  const [sortValue, setSortValue] = useState('modifiedAt-desc');
  const [gameFilter, setGameFilter] = useState<GameFilter>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);

  // Parse combined sort value into field and direction
  const [sortField, sortDir] = sortValue.split('-') as [string, 'asc' | 'desc'];
  
  // Fetch all games visible to the user (will filter client-side for workshop context)
  const { data: rawGames, isLoading, isFetching, error, refetch } = useGames({ 
    sortBy: sortField as 'name' | 'createdAt' | 'modifiedAt' | 'playCount' | 'visibility' | 'creator', 
    sortDir, 
    filter: 'all', // Fetch all visible games, filter client-side
    search: debouncedSearch || undefined 
  });
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const deleteGame = useDeleteGame();
  const deleteSession = useDeleteSession();
  const exportGameYaml = useExportGameYaml();
  const importGameYaml = useImportGameYaml();
  const cloneGame = useCloneGame();
  
  const fileInputRef = React.useRef<HTMLInputElement>(null);

  // Client-side filtering based on selected filter
  const games = React.useMemo(() => {
    if (!rawGames) return [];
    
    switch (gameFilter) {
      case 'mine':
        return rawGames.filter(game => game.creatorId === currentUserId);
      case 'workshop':
        // Workshop games are those with a workshopId (excluding public-only games)
        return rawGames.filter(game => game.workshopId && !game.public);
      case 'public':
        return rawGames.filter(game => game.public);
      default:
        return rawGames;
    }
  }, [rawGames, gameFilter, currentUserId]);

  const isOwner = (game: ObjGame) => game.creatorId === currentUserId;
  
  const canEditGame = (game: ObjGame) => {
    // Owner can always edit their games
    if (isOwner(game)) return true;
    // Head/Staff can edit workshop games
    if (canEditAllWorkshopGames && game.workshopId) return true;
    return false;
  };

  const canDeleteGame = (game: ObjGame) => {
    // Only owner can delete their games
    return isOwner(game);
  };

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

  const handleEditGame = (game: ObjGame) => {
    if (game.id && canEditGame(game)) {
      setGameToView(game.id);
      setGameToViewReadOnly(false);
      openViewModal();
    }
  };

  const handleViewGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      setGameToViewReadOnly(!canEditGame(game));
      openViewModal();
    }
  };

  const handleCopyFromModal = async () => {
    if (!gameToView) return;
    try {
      const newGame = await cloneGame.mutateAsync(gameToView);
      closeViewModal();
      setGameToView(null);
      // Open the new game for editing
      if (newGame.id) {
        setGameToView(newGame.id);
        setGameToViewReadOnly(false);
        openViewModal();
      }
    } catch {
      // Error handled by mutation
    }
  };

  const handleDeleteClick = (game: ObjGame) => {
    if (canDeleteGame(game)) {
      setGameToDelete(game);
      openDeleteModal();
    }
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

  const getDateLabel = (game: ObjGame) => {
    const dateValue = sortField === 'createdAt' ? game.meta?.createdAt : game.meta?.modifiedAt;
    return dateValue ? new Date(dateValue).toLocaleDateString() : undefined;
  };

  const getCardActions = (game: ObjGame): GameCardAction[] => {
    const actions: GameCardAction[] = [];
    
    if (canEditGame(game)) {
      actions.push({
        key: 'edit',
        icon: null,
        label: t('editGame'),
        onClick: () => handleEditGame(game),
      });
    }
    
    actions.push({
      key: 'copy',
      icon: <IconCopy size={16} />,
      label: t('copyGame'),
      onClick: () => handleCopyGame(game),
    });
    
    actions.push({
      key: 'export',
      icon: <IconDownload size={16} />,
      label: t('games.importExport.exportButton'),
      onClick: () => handleExport(game),
    });
    
    if (canDeleteGame(game)) {
      actions.push({
        key: 'delete',
        icon: null,
        label: t('deleteGame'),
        onClick: () => handleDeleteClick(game),
      });
    }
    
    return actions;
  };

  const getGameBadge = (game: ObjGame) => {
    if (isOwner(game)) {
      return (
        <Badge size="xs" color="violet" variant="light" leftSection={<IconUser size={10} />}>
          {tWorkshop('filters.mine')}
        </Badge>
      );
    }
    if (game.public) {
      return (
        <Badge size="xs" color="green" variant="light" leftSection={<IconWorld size={10} />}>
          {t('games.visibility.public')}
        </Badge>
      );
    }
    if (game.workshopId) {
      return (
        <Badge size="xs" color="accent" variant="light" leftSection={<IconSchool size={10} />}>
          {tWorkshop('filters.workshop')}
        </Badge>
      );
    }
    return null;
  };

  const columns: DataTableColumn<ObjGame>[] = [
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
      key: 'creator',
      header: t('games.fields.creator'),
      width: 150,
      render: (game) => (
        <Text size="sm" c="gray.6" lineClamp={1}>
          {isOwner(game) ? tWorkshop('you') : game.creatorName || '-'}
        </Text>
      ),
    },
    {
      key: 'type',
      header: tWorkshop('gameType'),
      width: 130,
      render: (game) => getGameBadge(game),
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
      width: 280,
      render: (game) => {
        const { hasSession, session } = getGameSessionState(game);
        return (
          <Group gap="xs" onClick={(e) => e.stopPropagation()} wrap="nowrap">
            {hasSession && session ? (
              <PlayGameButton onClick={() => handleContinueGame(session)} size="xs">
                {t('myGames.continue')}
              </PlayGameButton>
            ) : (
              <PlayGameButton onClick={() => handlePlayGame(game)} size="xs">
                {t('myGames.play')}
              </PlayGameButton>
            )}
            {canEditGame(game) ? (
              <Tooltip label={t('editGame')} withArrow>
                <EditIconButton onClick={() => handleEditGame(game)} aria-label={t('edit')} />
              </Tooltip>
            ) : (
              <Tooltip label={t('viewGame')} withArrow>
                <GenericIconButton
                  icon={<IconEye size={16} />}
                  onClick={() => handleViewGame(game)}
                  aria-label={t('viewGame')}
                />
              </Tooltip>
            )}
            <Tooltip label={t('copyGame')} withArrow>
              <GenericIconButton
                icon={<IconCopy size={16} />}
                onClick={() => handleCopyGame(game)}
                aria-label={t('copyGame')}
              />
            </Tooltip>
            {canDeleteGame(game) && (
              <Tooltip label={t('deleteGame')} withArrow>
                <DeleteIconButton onClick={() => handleDeleteClick(game)} aria-label={t('delete')} />
              </Tooltip>
            )}
          </Group>
        );
      },
    },
  ];

  const sortOptions: SortOption[] = [
    { value: 'modifiedAt-desc', label: t('games.sort.modifiedAt-desc') },
    { value: 'modifiedAt-asc', label: t('games.sort.modifiedAt-asc') },
    { value: 'createdAt-desc', label: t('games.sort.createdAt-desc') },
    { value: 'createdAt-asc', label: t('games.sort.createdAt-asc') },
    { value: 'name-asc', label: t('games.sort.name-asc') },
    { value: 'name-desc', label: t('games.sort.name-desc') },
  ];

  const filterOptions = [
    { value: 'all', label: tWorkshop('filters.all') },
    { value: 'mine', label: tWorkshop('filters.mine') },
    { value: 'workshop', label: tWorkshop('filters.workshop') },
    { value: 'public', label: tWorkshop('filters.public') },
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
      <Alert icon={<IconAlertCircle size={16} />} title={t('error')} color="red">
        {t('games.errors.loadFailed')}
      </Alert>
    );
  }

  return (
    <>
      <Stack gap="lg" h={{ base: 'calc(100vh - 180px)', sm: 'calc(100vh - 280px)' }} style={{ overflow: 'hidden' }}>
        {/* Header section */}
        <Stack gap="md" style={{ flexShrink: 0 }}>
          <PageTitle>{tWorkshop('title')}</PageTitle>
          
          {/* Workshop info */}
          <Group gap="md">
            {workshopName && (
              <Badge size="lg" color="accent" variant="light" leftSection={<IconSchool size={14} />}>
                {workshopName}
              </Badge>
            )}
            {organizationName && (
              <Text size="sm" c="dimmed">{organizationName}</Text>
            )}
          </Group>

          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileSelect}
            accept=".yaml,.yml"
            style={{ display: 'none' }}
          />

          <Group justify="space-between" wrap="wrap" gap="sm">
            <Group gap="sm">
              <ActionButton onClick={openCreateModal}>
                {t('games.createButton')}
              </ActionButton>
              <ActionButton onClick={handleImportClick}>
                {t('games.importExport.importButton')}
              </ActionButton>
              <ExpandableSearch
                value={searchQuery}
                onChange={setSearchQuery}
                placeholder={t('search')}
              />
            </Group>
            <Group gap="sm" wrap="wrap">
              <FilterSegmentedControl
                value={gameFilter}
                onChange={(val) => setGameFilter(val as GameFilter)}
                options={filterOptions}
              />
              {(rawGames?.length ?? 0) > 0 && (
                <SortSelector 
                  options={sortOptions} 
                  value={sortValue} 
                  onChange={setSortValue}
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
                      {tWorkshop('empty.title')}
                    </Text>
                    <Text size="sm" c="gray.5" ta="center">
                      {tWorkshop('empty.description')}
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
                        showCreator
                        isOwner={isOwner(game)}
                        creatorLabel={tWorkshop('you')}
                        actions={getCardActions(game)}
                        dateLabel={getDateLabel(game)}
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
                    {tWorkshop('empty.title')}
                  </Text>
                  <Text size="sm" c="gray.5" ta="center">
                    {tWorkshop('empty.description')}
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
                      showCreator
                      isOwner={isOwner(game)}
                      creatorLabel={tWorkshop('you')}
                      actions={getCardActions(game)}
                      dateLabel={getDateLabel(game)}
                    />
                  );
                }}
                emptyState={
                  <DataTableEmptyState
                    icon={<IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
                    title={tWorkshop('empty.title')}
                    description={tWorkshop('empty.description')}
                  />
                }
              />
            )}
          </DimmedLoader>
        </Box>
      </Stack>

      <GameEditModal
        opened={createModalOpened}
        onClose={closeCreateModal}
        onCreate={handleCreateGame}
        createLoading={createGame.isPending}
      />

      <GameEditModal
        gameId={gameToView}
        opened={viewModalOpened}
        onClose={() => {
          closeViewModal();
          setGameToView(null);
        }}
        readOnly={gameToViewReadOnly}
        onCopy={gameToViewReadOnly ? handleCopyFromModal : undefined}
        copyLoading={cloneGame.isPending}
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
