import { useState, useMemo } from 'react';
import {
  Container,
  Stack,
  Group,
  Alert,
  SegmentedControl,
  Badge,
  Text,
  Tooltip,
} from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import {
  IconAlertCircle,
  IconMoodEmpty,
  IconWorld,
  IconLock,
  IconCopy,
  IconStar,
  IconStarFilled,
  IconFilter,
} from '@tabler/icons-react';
import { PageTitle } from '@components/typography';
import { SortSelector, type SortOption } from '@components/controls';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { PlayIconButton, GenericIconButton } from '@components/buttons';
import { useGames } from '@/api/hooks';
import { useAuth } from '@/providers/AuthProvider';
import type { ObjGame } from '@/api/generated';
import { filterGames, sortGames, type GameFilter, type GameSortField } from '../types';
import { PlayGameCard } from './PlayGameCard';

interface PlayGamesListProps {
  onPlay: (game: ObjGame) => void;
  onClone: (game: ObjGame) => void;
  isCloning?: boolean;
}

export function PlayGamesList({ onPlay, onClone }: PlayGamesListProps) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');
  const { backendUser } = useAuth();

  const [filter, setFilter] = useState<GameFilter>('all');
  const [sortField, setSortField] = useState<GameSortField>('modifiedAt');
  const [favorites, setFavorites] = useState<Set<string>>(new Set());

  const { data: games, isLoading, error } = useGames();

  const filteredAndSortedGames = useMemo(() => {
    if (!games) return [];
    const filtered = filterGames(games, filter, backendUser?.id);
    return sortGames(filtered, { field: sortField, direction: 'desc' });
  }, [games, filter, sortField, backendUser?.id]);

  const handleToggleFavorite = (game: ObjGame) => {
    if (!game.id) return;
    setFavorites((prev) => {
      const next = new Set(prev);
      if (next.has(game.id!)) {
        next.delete(game.id!);
      } else {
        next.add(game.id!);
      }
      return next;
    });
  };

  const isOwner = (game: ObjGame) => game.meta?.createdBy === backendUser?.id;
  const isFavorite = (game: ObjGame) => game.id ? favorites.has(game.id) : false;

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString();
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
            {isOwner(game) && (
              <Badge size="xs" color="accent" variant="light">
                {t('play.owner')}
              </Badge>
            )}
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
      key: 'modified',
      header: t('games.fields.modified'),
      width: 120,
      hideOnMobile: true,
      render: (game) => (
        <Text size="sm" c="gray.5">
          {formatDate(game.meta?.modifiedAt)}
        </Text>
      ),
    },
    {
      key: 'actions',
      header: t('actions'),
      width: 140,
      render: (game) => (
        <Group gap="xs" wrap="nowrap" onClick={(e) => e.stopPropagation()}>
          <Tooltip label={t('play.playGame')} position="top" withArrow>
            <PlayIconButton
              onClick={() => onPlay(game)}
              aria-label={t('play.playGame')}
            />
          </Tooltip>
          <Tooltip label={isFavorite(game) ? t('play.unfavorite') : t('play.favorite')} position="top" withArrow>
            <GenericIconButton
              icon={isFavorite(game) ? <IconStarFilled size={16} /> : <IconStar size={16} />}
              variant="subtle"
              color={isFavorite(game) ? 'yellow' : 'gray'}
              onClick={() => handleToggleFavorite(game)}
              aria-label={isFavorite(game) ? t('play.unfavorite') : t('play.favorite')}
            />
          </Tooltip>
          <Tooltip label={t('play.cloneGame')} position="top" withArrow>
            <GenericIconButton
              icon={<IconCopy size={16} />}
              variant="subtle"
              color="gray"
              onClick={() => onClone(game)}
              aria-label={t('play.cloneGame')}
            />
          </Tooltip>
        </Group>
      ),
    },
  ];

  const filterOptions = [
    { value: 'all', label: t('play.filters.all') },
    { value: 'own', label: t('play.filters.own') },
    { value: 'public', label: t('play.filters.public') },
    { value: 'organization', label: t('play.filters.organization') },
    { value: 'favorites', label: t('play.filters.favorites') },
  ];

  const sortOptions: SortOption[] = [
    { value: 'modifiedAt', label: t('games.sort.modifiedAt') },
    { value: 'createdAt', label: t('games.sort.createdAt') },
    { value: 'name', label: t('games.sort.name') },
  ];

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
    <Container size="lg" py="xl">
      <Stack gap="lg">
        <PageTitle>{t('play.title')}</PageTitle>

        <Group justify="space-between" wrap="wrap" gap="sm">
          {isMobile ? (
            <Group gap="xs">
              <IconFilter size={16} color="var(--mantine-color-gray-6)" />
              <SegmentedControl
                size="xs"
                value={filter}
                onChange={(v) => setFilter(v as GameFilter)}
                data={filterOptions}
              />
            </Group>
          ) : (
            <SegmentedControl
              size="sm"
              value={filter}
              onChange={(v) => setFilter(v as GameFilter)}
              data={filterOptions}
            />
          )}
          <SortSelector
            options={sortOptions}
            value={sortField}
            onChange={(v) => setSortField(v as GameSortField)}
            label={t('games.sort.label')}
          />
        </Group>

        <DataTable
          data={filteredAndSortedGames}
          columns={columns}
          getRowKey={(game) => game.id || ''}
          onRowClick={onPlay}
          isLoading={isLoading}
          renderMobileCard={(game) => (
            <PlayGameCard
              game={game}
              isOwner={isOwner(game)}
              isFavorite={isFavorite(game)}
              onPlay={onPlay}
              onClone={onClone}
              onToggleFavorite={handleToggleFavorite}
            />
          )}
          emptyState={
            <DataTableEmptyState
              icon={<IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
              title={t('play.empty.title')}
              description={t('play.empty.description')}
            />
          }
        />
      </Stack>
    </Container>
  );
}
