import { Table, Group, Badge, Text, Stack } from '@mantine/core';
import { IconWorld, IconLock } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { PlayGameButton, EditButtonWithText, DeleteButtonWithText, ExportButtonWithText } from '@components/buttons';
import type { ObjGame } from '@/api/generated';

interface GamesTableProps {
  games: ObjGame[];
  onView: (game: ObjGame) => void;
  onEdit: (game: ObjGame) => void;
  onDelete: (game: ObjGame) => void;
  onExport: (game: ObjGame) => void;
  onPlay: (game: ObjGame) => void;
  fillHeight?: boolean;
}

export function GamesTable({ games, onView, onEdit, onDelete, onExport, onPlay, fillHeight = false }: GamesTableProps) {
  const { t } = useTranslation('common');

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString();
  };

  return (
    <Table.ScrollContainer minWidth={500} style={fillHeight ? { overflowY: 'auto', flex: 1 } : undefined}>
      <Table verticalSpacing="md" horizontalSpacing="lg" stickyHeader={fillHeight}>
        <Table.Thead>
          <Table.Tr style={{ borderBottom: '2px solid var(--mantine-color-gray-2)' }}>
            <Table.Th w={120} style={{ color: 'var(--mantine-color-gray-6)', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              {/* Play column - no header text */}
            </Table.Th>
            <Table.Th style={{ color: 'var(--mantine-color-gray-6)', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              {t('games.fields.name')}
            </Table.Th>
            <Table.Th style={{ color: 'var(--mantine-color-gray-6)', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              {t('games.fields.visibility')}
            </Table.Th>
            <Table.Th style={{ color: 'var(--mantine-color-gray-6)', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              {t('games.fields.modified')}
            </Table.Th>
            <Table.Th w={280} style={{ color: 'var(--mantine-color-gray-6)', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              {t('actions')}
            </Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {games.map((game) => (
            <Table.Tr 
              key={game.id}
              style={{ 
                cursor: 'pointer',
                transition: 'background-color 150ms ease',
              }}
              onClick={() => onView(game)}
            >
              <Table.Td onClick={(e) => e.stopPropagation()}>
                <PlayGameButton onClick={() => onPlay(game)}>
                  {t('games.playNow')}
                </PlayGameButton>
              </Table.Td>
              <Table.Td>
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
              </Table.Td>
              <Table.Td>
                {game.public ? (
                  <Badge size="sm" color="green" variant="light" leftSection={<IconWorld size={12} />}>
                    {t('games.visibility.public')}
                  </Badge>
                ) : (
                  <Badge size="sm" color="gray" variant="light" leftSection={<IconLock size={12} />}>
                    {t('games.visibility.private')}
                  </Badge>
                )}
              </Table.Td>
              <Table.Td>
                <Text size="sm" c="gray.5">{formatDate(game.meta?.modifiedAt)}</Text>
              </Table.Td>
              <Table.Td onClick={(e) => e.stopPropagation()}>
                <Group gap="xs" wrap="nowrap">
                  <EditButtonWithText onClick={() => onEdit(game)}>
                    {t('edit')}
                  </EditButtonWithText>
                  <ExportButtonWithText onClick={() => onExport(game)}>
                    {t('games.importExport.exportButton')}
                  </ExportButtonWithText>
                  <DeleteButtonWithText onClick={() => onDelete(game)}>
                    {t('delete')}
                  </DeleteButtonWithText>
                </Group>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Table.ScrollContainer>
  );
}
