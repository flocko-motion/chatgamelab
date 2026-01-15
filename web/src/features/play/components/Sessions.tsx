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
  TextInput,
} from '@mantine/core';
import { useMediaQuery, useDebouncedValue } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { IconPlus, IconAlertCircle, IconMoodEmpty, IconSearch } from '@tabler/icons-react';
import { TextButton, PlayGameButton, DeleteIconButton, DeleteButtonWithText } from '@components/buttons';
import { useModals } from '@mantine/modals';
import { SortSelector, type SortOption } from '@components/controls';
import { PageTitle } from '@components/typography';
import { DataTable, DataTableEmptyState, type DataTableColumn } from '@components/DataTable';
import { useUserSessions, useDeleteSession } from '@/api/hooks';
import type { DbUserSessionWithGame } from '@/api/generated';
import { ROUTES } from '@/common/routes/routes';

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return '';
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

interface SessionCardProps {
  session: DbUserSessionWithGame;
  onResume: (session: DbUserSessionWithGame) => void;
  onDelete: (session: DbUserSessionWithGame) => void;
  isDeleting?: boolean;
}

function SessionCard({ session, onResume, onDelete, isDeleting }: SessionCardProps) {
  const { t } = useTranslation('common');

  return (
    <Card shadow="sm" p="lg" radius="md" withBorder>
      <Stack gap="sm">
        <Group gap="md" align="flex-start" wrap="nowrap">
          <PlayGameButton onClick={() => onResume(session)} size="sm">
            {t('sessions.continueGame')}
          </PlayGameButton>
          <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
            <Group justify="space-between" wrap="nowrap">
              <Text fw={600} size="md" lineClamp={1}>
                {session.gameName || t('sessions.untitledGame')}
              </Text>
              <Badge size="sm" color="gray" variant="light">
                {session.aiModel || 'AI'}
              </Badge>
            </Group>
            <Text size="xs" c="dimmed">
              {formatRelativeTime(session.meta?.modifiedAt)}
            </Text>
          </Stack>
          <DeleteIconButton
            onClick={() => onDelete(session)}
            loading={isDeleting}
            aria-label={t('delete')}
          />
        </Group>
      </Stack>
    </Card>
  );
}

type SortField = 'lastPlayed' | 'game' | 'model';

export function Sessions() {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const modals = useModals();
  const isMobile = useMediaQuery('(max-width: 48em)');
  const [sortField, setSortField] = useState<SortField>('lastPlayed');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);
  const [deletingSessionId, setDeletingSessionId] = useState<string | null>(null);

  const { data: sessions, isLoading, error } = useUserSessions({
    search: debouncedSearch || undefined,
    sortBy: sortField,
  });

  const deleteSession = useDeleteSession();

  const handleResume = (session: DbUserSessionWithGame) => {
    if (session.id) {
      navigate({ to: `/sessions/${session.id}` as '/' });
    }
  };

  const handleStartNewGame = () => {
    navigate({ to: `${ROUTES.SESSIONS}/new` as '/' });
  };

  const handleDelete = (session: DbUserSessionWithGame) => {
    if (!session.id) return;
    
    modals.openConfirmModal({
      title: t('sessions.deleteConfirm.title'),
      children: (
        <Text size="sm">
          {t('sessions.deleteConfirm.message', { game: session.gameName || t('sessions.untitledGame') })}
        </Text>
      ),
      labels: {
        confirm: t('delete'),
        cancel: t('cancel'),
      },
      confirmProps: { color: 'red' },
      onConfirm: () => {
        setDeletingSessionId(session.id!);
        deleteSession.mutate(session.id!, {
          onSettled: () => setDeletingSessionId(null),
        });
      },
    });
  };

  const columns: DataTableColumn<DbUserSessionWithGame>[] = [
    {
      key: 'continue',
      header: '', // No header for continue column
      width: 140,
      render: (session) => (
        <div onClick={(e) => e.stopPropagation()}>
          <PlayGameButton onClick={() => handleResume(session)}>
            {t('sessions.continueGame')}
          </PlayGameButton>
        </div>
      ),
    },
    {
      key: 'gameName',
      header: t('sessions.fields.game'),
      render: (session) => (
        <Stack gap={2}>
          <Text fw={600} size="sm" lineClamp={1}>
            {session.gameName || t('sessions.untitledGame')}
          </Text>
          <Text size="xs" c="dimmed" lineClamp={1} maw={250}>
            {session.gameDescription || t('sessions.noDescription')}
          </Text>
        </Stack>
      ),
    },
    {
      key: 'model',
      header: t('sessions.fields.model'),
      width: 140,
      render: (session) => (
        <Badge size="sm" color="gray" variant="light">
          {session.aiModel || 'AI'}
        </Badge>
      ),
    },
    {
      key: 'lastPlayed',
      header: t('sessions.fields.lastPlayed'),
      width: 140,
      hideOnMobile: true,
      render: (session) => (
        <Text size="sm" c="dimmed">
          {formatRelativeTime(session.meta?.modifiedAt)}
        </Text>
      ),
    },
    {
      key: 'actions',
      header: t('actions'),
      width: 100,
      render: (session) => (
        <div onClick={(e) => e.stopPropagation()}>
          <DeleteButtonWithText
            onClick={() => handleDelete(session)}
            loading={deletingSessionId === session.id}
          >
            {t('delete')}
          </DeleteButtonWithText>
        </div>
      ),
    },
  ];

  const sortOptions: SortOption[] = [
    { value: 'lastPlayed', label: t('sessions.sort.lastPlayed') },
    { value: 'game', label: t('sessions.sort.game') },
    { value: 'model', label: t('sessions.sort.model') },
  ];

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
                    <Skeleton height={16} width="40%" />
                    <Skeleton height={32} width={80} />
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
          {t('sessions.errors.loadFailed')}
        </Alert>
      </Container>
    );
  }

  const hasSessions = (sessions?.length ?? 0) > 0;
  const isSearching = debouncedSearch.length > 0;
  const showEmptySearch = isSearching && !hasSessions;

  return (
    <Container size="lg" py="xl" h="calc(100vh - 210px)">
      <Stack gap="lg" h="100%">
        <PageTitle>{t('sessions.title')}</PageTitle>

        <Group justify="space-between" wrap="wrap" gap="sm">
          <TextButton
            leftSection={<IconPlus size={16} />}
            onClick={handleStartNewGame}
          >
            {t('sessions.startNewGame')}
          </TextButton>
          <Group gap="sm" wrap="wrap">
            <TextInput
              placeholder={t('search')}
              leftSection={<IconSearch size={16} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.currentTarget.value)}
              size={isMobile ? 'xs' : 'sm'}
              w={{ base: 150, sm: 200 }}
            />
            <SortSelector
              options={sortOptions}
              value={sortField}
              onChange={(v) => setSortField(v as SortField)}
              label={t('sessions.sort.label')}
            />
          </Group>
        </Group>

        {showEmptySearch ? (
          <Card shadow="sm" p="xl" radius="md" withBorder>
            <Stack align="center" gap="md" py="xl">
              <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
              <Text c="gray.6" ta="center">
                {t('sessions.empty.noResults')}
              </Text>
            </Stack>
          </Card>
        ) : !hasSessions ? (
          <Card shadow="sm" p="xl" radius="md" withBorder>
            <Stack align="center" gap="md" py="xl">
              <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
              <Text c="gray.6" ta="center">
                {t('sessions.empty.title')}
              </Text>
              <Text size="sm" c="gray.5" ta="center">
                {t('sessions.empty.description')}
              </Text>
            </Stack>
          </Card>
        ) : isMobile ? (
          <SimpleGrid cols={1} spacing="md">
            {sessions?.map((session) => (
              <SessionCard
                key={session.id}
                session={session}
                onResume={handleResume}
                onDelete={handleDelete}
                isDeleting={deletingSessionId === session.id}
              />
            ))}
          </SimpleGrid>
        ) : (
          <DataTable
            data={sessions ?? []}
            columns={columns}
            getRowKey={(session) => session.id || ''}
            onRowClick={handleResume}
            isLoading={false}
            fillHeight
            renderMobileCard={(session) => (
              <SessionCard
                session={session}
                onResume={handleResume}
                onDelete={handleDelete}
                isDeleting={deletingSessionId === session.id}
              />
            )}
            emptyState={
              <DataTableEmptyState
                icon={<IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
                title={t('sessions.empty.title')}
                description={t('sessions.empty.description')}
              />
            }
          />
        )}
      </Stack>
    </Container>
  );
}
