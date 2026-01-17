import { useState } from 'react';
import {
  Stack,
  Group,
  Card,
  Alert,
  SimpleGrid,
  Skeleton,
  Text,
  Badge,
  TextInput,
  Tooltip,
} from '@mantine/core';
import { useMediaQuery, useDebouncedValue } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { useModals } from '@mantine/modals';
import {
  IconAlertCircle,
  IconMoodEmpty,
  IconSearch,
  IconCopy,
  IconStar,
  IconStarFilled,
} from '@tabler/icons-react';
import { PageTitle } from '@components/typography';
import { SortSelector, type SortOption, FilterSegmentedControl } from '@components/controls';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { DimmedLoader } from '@components/LoadingAnimation';
import { PlayGameButton, TextButton, GenericIconButton } from '@components/buttons';
import { useGames, useGameSessionMap, useDeleteSession, useCloneGame, useFavoriteGames, useAddFavorite, useRemoveFavorite } from '@/api/hooks';
import { useAuth } from '@/providers/AuthProvider';
import type { ObjGame, DbUserSessionWithGame } from '@/api/generated';
import { type GameFilter, type GameSortField } from '@/features/play/types';

export function AllGames() {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const modals = useModals();
  const isMobile = useMediaQuery('(max-width: 48em)');
  const { backendUser } = useAuth();

  const [filter, setFilter] = useState<GameFilter>('all');
  const [sortField, setSortField] = useState<GameSortField>('modifiedAt');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);

  // For favorites filter, we fetch all games and filter client-side using the favorites list
  const apiFilter = filter === 'favorites' ? 'all' : filter;
  
  const { data: rawGames, isLoading, isFetching, error } = useGames({
    search: debouncedSearch || undefined,
    sortBy: sortField,
    sortDir: 'desc',
    filter: apiFilter,
  });

  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const deleteSession = useDeleteSession();
  const cloneGame = useCloneGame();
  const { data: favoriteGames } = useFavoriteGames();
  const addFavorite = useAddFavorite();
  const removeFavorite = useRemoveFavorite();

  const favoriteGameIds = new Set(favoriteGames?.map(g => g.id) ?? []);
  
  // Apply client-side favorites filter
  const games = filter === 'favorites' 
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

  const isOwner = (game: ObjGame) => {
    if (!backendUser?.id || !game.creatorId) return false;
    return game.creatorId === backendUser.id;
  };

  const getGameSessionState = (game: ObjGame) => {
    if (!game.id) return { hasSession: false, session: undefined };
    const session = sessionMap.get(game.id);
    return { hasSession: !!session, session };
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
      title: t('allGames.restartConfirm.title'),
      children: (
        <Text size="sm">
          {t('allGames.restartConfirm.message', { game: game.name || t('sessions.untitledGame') })}
        </Text>
      ),
      labels: {
        confirm: t('allGames.restartConfirm.confirm'),
        cancel: t('cancel'),
      },
      confirmProps: { color: 'red' },
      onConfirm: async () => {
        await deleteSession.mutateAsync(session.id!);
        navigate({ to: '/games/$gameId/play', params: { gameId: game.id! } });
      },
    });
  };

  const handleCopyGame = (game: ObjGame) => {
    if (!game.id) return;
    
    modals.openConfirmModal({
      title: t('allGames.copyConfirm.title'),
      children: (
        <Text size="sm">
          {t('allGames.copyConfirm.message', { name: game.name || t('sessions.untitledGame') })}
        </Text>
      ),
      labels: {
        confirm: t('allGames.copyConfirm.confirm'),
        cancel: t('cancel'),
      },
      onConfirm: async () => {
        const newGame = await cloneGame.mutateAsync(game.id!);
        if (newGame.id) {
          navigate({ to: `/my-games/${newGame.id}` as '/' });
        }
      },
    });
  };

  const renderPlayButton = (game: ObjGame) => {
    const { hasSession, session } = getGameSessionState(game);
    
    if (!hasSession) {
      return (
        <PlayGameButton onClick={() => handlePlayGame(game)} size="xs" style={{ width: '100%' }}>
          {t('allGames.play')}
        </PlayGameButton>
      );
    }
    
    return (
      <Stack gap={4}>
        <PlayGameButton onClick={() => handleContinueGame(session!)} size="xs" style={{ width: '100%' }}>
          {t('allGames.continue')}
        </PlayGameButton>
        <TextButton onClick={() => handleRestartGame(game, session!)} size="xs">
          {t('allGames.restart')}
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
          <Tooltip label={isFavorite(game) ? t('allGames.unfavorite') : t('allGames.favorite')} withArrow>
            <GenericIconButton
              icon={isFavorite(game) ? <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" /> : <IconStar size={18} />}
              onClick={() => handleToggleFavorite(game)}
              aria-label={isFavorite(game) ? t('allGames.unfavorite') : t('allGames.favorite')}
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
          <Text fw={600} size="sm" c="gray.8" lineClamp={1}>
            {game.name}
          </Text>
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
      width: 120,
      render: (game) => (
        isOwner(game) ? (
          <Badge size="sm" color="violet" variant="light">
            {t('games.fields.me')}
          </Badge>
        ) : (
          <Text size="sm" c="gray.6" lineClamp={1}>
            {game.creatorName || '-'}
          </Text>
        )
      ),
    },
    {
      key: 'playCount',
      header: t('games.fields.playCount'),
      width: 80,
      render: (game) => (
        <Text size="sm" c="gray.6" ta="center">
          {game.playCount ?? 0}
        </Text>
      ),
    },
    {
      key: 'date',
      header: sortField === 'createdAt' ? t('games.fields.created') : t('games.fields.modified'),
      width: 100,
      render: (game) => {
        const dateValue = sortField === 'createdAt' ? game.meta?.createdAt : game.meta?.modifiedAt;
        return (
          <Text size="sm" c="gray.6">
            {dateValue ? new Date(dateValue).toLocaleDateString() : '-'}
          </Text>
        );
      },
    },
    {
      key: 'actions',
      header: '',
      width: 60,
      render: (game) => (
        <div onClick={(e) => e.stopPropagation()}>
          <Tooltip label={t('allGames.copyGame')} withArrow>
            <GenericIconButton
              icon={<IconCopy size={16} />}
              onClick={() => handleCopyGame(game)}
              aria-label={t('allGames.copyGame')}
            />
          </Tooltip>
        </div>
      ),
    },
  ];

  const filterOptions = [
    { value: 'all', label: t('allGames.filters.all') },
    { value: 'favorites', label: t('allGames.filters.favorites') },
    { value: 'own', label: t('allGames.filters.own') },
    { value: 'public', label: t('allGames.filters.public') },
  ];

  const sortOptions: SortOption[] = [
    { value: 'modifiedAt', label: t('games.sort.modifiedAt') },
    { value: 'createdAt', label: t('games.sort.createdAt') },
    { value: 'name', label: t('games.sort.name') },
    { value: 'playCount', label: t('games.sort.playCount') },
    { value: 'creator', label: t('games.sort.creator') },
  ];

  const hasData = rawGames !== undefined;
  const isInitialLoading = !hasData && (isLoading || sessionsLoading);
  const isRefetching = isFetching && hasData;

  if (isInitialLoading) {
    return (
      <Stack gap="xl">
        <Skeleton height={40} width="50%" />
        <Skeleton height={36} width={300} />
        <Skeleton height={300} />
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
    <Stack gap="lg" h="calc(100vh - 280px)">
        <PageTitle>{t('allGames.title')}</PageTitle>

        <Group justify="space-between" wrap="wrap" gap="sm">
          <TextInput
            placeholder={t('search')}
            leftSection={<IconSearch size={16} />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.currentTarget.value)}
            size={isMobile ? 'xs' : 'sm'}
            w={{ base: 150, sm: 200 }}
          />
          <Group gap="sm" wrap="wrap">
            <FilterSegmentedControl
              value={filter}
              onChange={setFilter}
              options={filterOptions}
            />
            <SortSelector
              options={sortOptions}
              value={sortField}
              onChange={(v) => setSortField(v as GameSortField)}
              label={t('games.sort.label')}
            />
          </Group>
        </Group>

        <DimmedLoader visible={isRefetching} loaderSize="lg">
        {isMobile ? (
            (games?.length ?? 0) === 0 ? (
              <Card shadow="sm" p="xl" radius="md" withBorder>
                <Stack align="center" gap="md" py="xl">
                  <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
                  <Text c="gray.6" ta="center">
                    {t('allGames.empty.title')}
                  </Text>
                  <Text size="sm" c="gray.5" ta="center">
                    {t('allGames.empty.description')}
                  </Text>
                </Stack>
              </Card>
            ) : (
              <SimpleGrid cols={1} spacing="md">
              {games?.map((game) => (
                <Card key={game.id} shadow="sm" p="lg" radius="md" withBorder>
                  <Stack gap="sm">
                    <Group gap="md" align="flex-start" wrap="nowrap">
                      <div onClick={(e) => e.stopPropagation()}>
                        <Tooltip label={isFavorite(game) ? t('allGames.unfavorite') : t('allGames.favorite')} withArrow>
                          <GenericIconButton
                            icon={isFavorite(game) ? <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" /> : <IconStar size={18} />}
                            onClick={() => handleToggleFavorite(game)}
                            aria-label={isFavorite(game) ? t('allGames.unfavorite') : t('allGames.favorite')}
                          />
                        </Tooltip>
                      </div>
                      <div onClick={(e) => e.stopPropagation()}>
                        {renderPlayButton(game)}
                      </div>
                      <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                        <Group gap="xs" wrap="nowrap">
                          <Text fw={600} size="md" lineClamp={1} style={{ flex: 1 }}>
                            {game.name}
                          </Text>
                          {isOwner(game) ? (
                            <Badge size="xs" color="violet" variant="light">
                              {t('games.fields.me')}
                            </Badge>
                          ) : (
                            game.creatorName && (
                              <Text size="xs" c="gray.6">
                                {game.creatorName}
                              </Text>
                            )
                          )}
                        </Group>
                        {game.description && (
                          <Text size="sm" c="gray.6" lineClamp={2}>
                            {game.description}
                          </Text>
                        )}
                      </Stack>
                      <div onClick={(e) => e.stopPropagation()}>
                        <Tooltip label={t('allGames.copyGame')} withArrow>
                          <GenericIconButton
                            icon={<IconCopy size={16} />}
                            onClick={() => handleCopyGame(game)}
                            aria-label={t('allGames.copyGame')}
                          />
                        </Tooltip>
                      </div>
                    </Group>
                  </Stack>
                </Card>
              ))}
            </SimpleGrid>
          )
        ) : (
          <DataTable
            data={games ?? []}
            columns={columns}
            getRowKey={(game) => game.id || ''}
            onRowClick={handlePlayGame}
            isLoading={false}
            fillHeight
            renderMobileCard={(game) => (
              <Card shadow="sm" p="lg" radius="md" withBorder>
                <Stack gap="sm">
                  <Group gap="md" align="flex-start" wrap="nowrap">
                    <div onClick={(e) => e.stopPropagation()}>
                      <Tooltip label={isFavorite(game) ? t('allGames.unfavorite') : t('allGames.favorite')} withArrow>
                        <GenericIconButton
                          icon={isFavorite(game) ? <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" /> : <IconStar size={18} />}
                          onClick={() => handleToggleFavorite(game)}
                          aria-label={isFavorite(game) ? t('allGames.unfavorite') : t('allGames.favorite')}
                        />
                      </Tooltip>
                    </div>
                    <div onClick={(e) => e.stopPropagation()}>
                      {renderPlayButton(game)}
                    </div>
                    <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                      <Group gap="xs" wrap="nowrap">
                        <Text fw={600} size="md" lineClamp={1} style={{ flex: 1 }}>
                          {game.name}
                        </Text>
                        {isOwner(game) ? (
                          <Badge size="xs" color="violet" variant="light">
                            {t('games.fields.me')}
                          </Badge>
                        ) : (
                          game.creatorName && (
                            <Text size="xs" c="gray.6">
                              {game.creatorName}
                            </Text>
                          )
                        )}
                      </Group>
                      {game.description && (
                        <Text size="sm" c="gray.6" lineClamp={2}>
                          {game.description}
                        </Text>
                      )}
                    </Stack>
                    <div onClick={(e) => e.stopPropagation()}>
                      <Tooltip label={t('allGames.copyGame')} withArrow>
                        <GenericIconButton
                          icon={<IconCopy size={16} />}
                          onClick={() => handleCopyGame(game)}
                          aria-label={t('allGames.copyGame')}
                        />
                      </Tooltip>
                    </div>
                  </Group>
                </Stack>
              </Card>
            )}
            emptyState={
              <DataTableEmptyState
                icon={<IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
                title={t('allGames.empty.title')}
                description={t('allGames.empty.description')}
              />
            }
          />
        )}
        </DimmedLoader>
    </Stack>
  );
}
