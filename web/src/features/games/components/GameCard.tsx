import { Card, Group, Badge, Stack, Text, Title, Box, Tooltip } from '@mantine/core';
import { IconWorld, IconLock, IconDownload } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { PlayGameButton, EditIconButton, DeleteIconButton, GenericIconButton } from '@components/buttons';
import type { ObjGame } from '@/api/generated';

interface GameCardProps {
  game: ObjGame;
  onView: (game: ObjGame) => void;
  onEdit: (game: ObjGame) => void;
  onDelete: (game: ObjGame) => void;
  onExport: (game: ObjGame) => void;
  onPlay: (game: ObjGame) => void;
}

export function GameCard({ game, onView, onEdit, onDelete, onExport, onPlay }: GameCardProps) {
  const { t } = useTranslation('common');

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString();
  };

  return (
    <Card 
      shadow="sm" 
      p="lg" 
      radius="md" 
      withBorder
      style={{ cursor: 'pointer', transition: 'box-shadow 150ms ease' }}
      onClick={() => onView(game)}
    >
      <Stack gap="sm">
        <Group gap="md" align="flex-start" wrap="nowrap">
          <Box onClick={(e) => e.stopPropagation()}>
            <PlayGameButton onClick={() => onPlay(game)} size="sm">
              {t('games.playNow')}
            </PlayGameButton>
          </Box>
          
          <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
            <Group gap="xs" wrap="nowrap">
              <Box style={{ flex: 1, minWidth: 0 }}>
                <Title order={4} lineClamp={1}>
                  {game.name}
                </Title>
              </Box>
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
        
        <Group justify="space-between" align="center">
          <Stack gap={2}>
            <Text size="xs" c="gray.5">{t('games.fields.modified')}</Text>
            <Text size="sm" c="gray.6">{formatDate(game.meta?.modifiedAt)}</Text>
          </Stack>
          <Group gap="xs" onClick={(e) => e.stopPropagation()}>
            <Tooltip label={t('edit')} position="top" withArrow>
              <span>
                <EditIconButton
                  onClick={() => onEdit(game)}
                  aria-label={t('edit')}
                />
              </span>
            </Tooltip>
            <Tooltip label={t('games.importExport.exportButton')} position="top" withArrow>
              <span>
                <GenericIconButton
                  icon={<IconDownload size={16} />}
                  onClick={() => onExport(game)}
                  aria-label={t('games.importExport.exportButton')}
                />
              </span>
            </Tooltip>
            <Tooltip label={t('delete')} position="top" withArrow>
              <span>
                <DeleteIconButton
                  onClick={() => onDelete(game)}
                  aria-label={t('delete')}
                />
              </span>
            </Tooltip>
          </Group>
        </Group>
      </Stack>
    </Card>
  );
}
