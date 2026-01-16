import { useState } from 'react';
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
  SegmentedControl,
  TextInput,
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
} from '@tabler/icons-react';
import { PageTitle } from '@components/typography';
import { SortSelector, type SortOption } from '@components/controls';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { PlayGameButton, TextButton, GenericIconButton } from '@components/buttons';
import { useGames, useGameSessionMap, useDeleteSession, useCloneGame, useUsers } from '@/api/hooks';
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

  const { data: games, isLoading, error } = useGames({
    search: debouncedSearch || undefined,
    sortBy: sortField,
    sortDir: 'desc',
    filter: filter,
  });

  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const deleteSession = useDeleteSession();
  const cloneGame = useCloneGame();
  const { data: users } = useUsers();

  const getUserName = (userId?: string) => {
    if (!userId || !users) return null;
    const user = users.find(u => u.id === userId);
    return user?.name || null;
  };

  const isOwner = (game: ObjGame) => {
    const createdBy = game.meta?.createdBy;
    if (!createdBy?.valid || !createdBy?.uuid || !backendUser?.id) return false;
    return createdBy.uuid.toLowerCase() === backendUser.id.toLowerCase();
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
      header: t('allGames.creator'),
      width: 120,
      render: (game) => (
        isOwner(game) ? (
          <Badge size="xs" color="accent" variant="light">
            {t('allGames.owner')}
          </Badge>
        ) : (
          <Text size="sm" c="dimmed">
            {getUserName(game.meta?.createdBy?.uuid) || '-'}
          </Text>
        )
      ),
    },
    {
      key: 'actions',
      header: '',
      width: 60,
      render: (game) => (
        <div onClick={(e) => e.stopPropagation()}>
          <GenericIconButton
            icon={<IconCopy size={16} />}
            onClick={() => handleCopyGame(game)}
            aria-label={t('allGames.copyGame')}
          />
        </div>
      ),
    },
  ];

  const filterOptions = [
    { value: 'all', label: t('allGames.filters.all') },
    { value: 'own', label: t('allGames.filters.own') },
    { value: 'public', label: t('allGames.filters.public') },
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
          <Skeleton height={36} width={300} />
          <Skeleton height={300} />
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
        <PageTitle>{t('allGames.title')}</PageTitle>

        <Group justify="space-between" wrap="wrap" gap="sm">
          <Group gap="sm" wrap="wrap">
            <TextInput
              placeholder={t('search')}
              leftSection={<IconSearch size={16} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.currentTarget.value)}
              size={isMobile ? 'xs' : 'sm'}
              w={{ base: 150, sm: 200 }}
            />
            <SegmentedControl
              size={isMobile ? 'xs' : 'sm'}
              value={filter}
              onChange={(v) => setFilter(v as GameFilter)}
              data={filterOptions}
            />
          </Group>
          <SortSelector
            options={sortOptions}
            value={sortField}
            onChange={(v) => setSortField(v as GameSortField)}
            label={t('games.sort.label')}
          />
        </Group>

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
                        {renderPlayButton(game)}
                      </div>
                      <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                        <Group gap="xs" wrap="nowrap">
                          <Text fw={600} size="md" lineClamp={1} style={{ flex: 1 }}>
                            {game.name}
                          </Text>
                          {isOwner(game) ? (
                            <Badge size="xs" color="accent" variant="light">
                              {t('allGames.owner')}
                            </Badge>
                          ) : (
                            getUserName(game.meta?.createdBy?.uuid) && (
                              <Text size="xs" c="dimmed">
                                {getUserName(game.meta?.createdBy?.uuid)}
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
                        <GenericIconButton
                          icon={<IconCopy size={16} />}
                          onClick={() => handleCopyGame(game)}
                          aria-label={t('allGames.copyGame')}
                        />
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
                      {renderPlayButton(game)}
                    </div>
                    <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                      <Group gap="xs" wrap="nowrap">
                        <Text fw={600} size="md" lineClamp={1} style={{ flex: 1 }}>
                          {game.name}
                        </Text>
                        {isOwner(game) ? (
                          <Badge size="xs" color="accent" variant="light">
                            {t('allGames.owner')}
                          </Badge>
                        ) : (
                          getUserName(game.meta?.createdBy?.uuid) && (
                            <Text size="xs" c="dimmed">
                              {getUserName(game.meta?.createdBy?.uuid)}
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
                      <GenericIconButton
                        icon={<IconCopy size={16} />}
                        onClick={() => handleCopyGame(game)}
                        aria-label={t('allGames.copyGame')}
                      />
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
      </Stack>
    </Container>
  );
}
