import { Card, Group, Badge, Stack, Text, Title, Box, Tooltip } from '@mantine/core';
import { IconWorld, IconLock, IconCopy, IconStar, IconStarFilled } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { PlayGameButton, GenericIconButton } from '@components/buttons';
import type { ObjGame } from '@/api/generated';

interface PlayGameCardProps {
  game: ObjGame;
  isOwner: boolean;
  isFavorite?: boolean;
  onPlay: (game: ObjGame) => void;
  onClone: (game: ObjGame) => void;
  onToggleFavorite?: (game: ObjGame) => void;
}

export function PlayGameCard({
  game,
  isOwner,
  isFavorite = false,
  onPlay,
  onClone,
  onToggleFavorite,
}: PlayGameCardProps) {
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
      onClick={() => onPlay(game)}
    >
      <Stack gap="sm">
        <Group gap="md" align="flex-start" wrap="nowrap">
          <Box onClick={(e) => e.stopPropagation()}>
            <PlayGameButton onClick={() => onPlay(game)} size="sm">
              {t('play.playNow')}
            </PlayGameButton>
          </Box>
          
          <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
            <Group gap="xs" wrap="nowrap">
              <Box style={{ flex: 1, minWidth: 0 }}>
                <Title order={4}>
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
              {isOwner && (
                <Badge size="sm" color="accent" variant="light">
                  {t('play.owner')}
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
          <Text size="xs" c="gray.5">
            {t('games.fields.modified')}: {formatDate(game.meta?.modifiedAt)}
          </Text>
          <Group gap="xs" onClick={(e) => e.stopPropagation()}>
            {onToggleFavorite && (
              <Tooltip label={isFavorite ? t('play.unfavorite') : t('play.favorite')} position="top" withArrow>
                <span>
                  <GenericIconButton
                    icon={isFavorite ? <IconStarFilled size={16} /> : <IconStar size={16} />}
                    variant="subtle"
                    color={isFavorite ? 'yellow' : 'gray'}
                    onClick={() => onToggleFavorite(game)}
                    aria-label={isFavorite ? t('play.unfavorite') : t('play.favorite')}
                  />
                </span>
              </Tooltip>
            )}
            <Tooltip label={t('play.cloneGame')} position="top" withArrow>
              <span>
                <GenericIconButton
                  icon={<IconCopy size={16} />}
                  variant="subtle"
                  color="gray"
                  onClick={() => onClone(game)}
                  aria-label={t('play.cloneGame')}
                />
              </span>
            </Tooltip>
          </Group>
        </Group>
      </Stack>
    </Card>
  );
}
