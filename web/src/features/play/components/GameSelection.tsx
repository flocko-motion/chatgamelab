import { useState } from 'react';
import {
  Container,
  Stack,
  Group,
  Alert,
  SegmentedControl,
  Badge,
  Text,
  TextInput,
} from '@mantine/core';
import { useMediaQuery, useDebouncedValue } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import {
  IconAlertCircle,
  IconMoodEmpty,
  IconWorld,
  IconLock,
  IconArrowLeft,
  IconSearch,
} from '@tabler/icons-react';
import { PageTitle } from '@components/typography';
import { SortSelector, type SortOption } from '@components/controls';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { PlayGameButton, TextButton } from '@components/buttons';
import { useGames } from '@/api/hooks';
import { useAuth } from '@/providers/AuthProvider';
import type { ObjGame } from '@/api/generated';
import { ROUTES } from '@/common/routes/routes';
import { type GameFilter, type GameSortField } from '../types';
import { PlayGameCard } from './PlayGameCard';

interface GameSelectionProps {
  onSelectGame: (game: ObjGame) => void;
}

export function GameSelection({ onSelectGame }: GameSelectionProps) {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
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

  const handleBack = () => {
    navigate({ to: ROUTES.SESSIONS as '/' });
  };

  const isOwner = (game: ObjGame) => game.meta?.createdBy === backendUser?.id;

  const columns: DataTableColumn<ObjGame>[] = [
    {
      key: 'play',
      header: '', // No header for play column
      width: 120,
      render: (game) => (
        <div onClick={(e) => e.stopPropagation()}>
          <PlayGameButton onClick={() => onSelectGame(game)}>
            {t('play.playNow')}
          </PlayGameButton>
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
  ];

  const filterOptions = [
    { value: 'all', label: t('play.filters.all') },
    { value: 'own', label: t('play.filters.own') },
    { value: 'public', label: t('play.filters.public') },
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
        <Group gap="md">
          <TextButton
            leftSection={<IconArrowLeft size={16} />}
            onClick={handleBack}
          >
            {t('back')}
          </TextButton>
        </Group>

        <PageTitle>{t('play.selectGameTitle')}</PageTitle>

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

        <DataTable
          data={games ?? []}
          columns={columns}
          getRowKey={(game) => game.id || ''}
          onRowClick={onSelectGame}
          isLoading={isLoading}
          renderMobileCard={(game) => (
            <PlayGameCard
              game={game}
              isOwner={isOwner(game)}
              isFavorite={false}
              onPlay={onSelectGame}
              onClone={() => {}}
              onToggleFavorite={() => {}}
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
