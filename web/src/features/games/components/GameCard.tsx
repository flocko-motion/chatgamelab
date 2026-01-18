import { Card, Group, Badge, Stack, Text, Box, Tooltip } from '@mantine/core';
import { IconWorld, IconLock, IconStar, IconStarFilled } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { PlayGameButton, TextButton, EditIconButton, DeleteIconButton, GenericIconButton } from '@components/buttons';
import type { ObjGame } from '@/api/generated';
import type { ReactNode } from 'react';

export interface GameCardAction {
  key: string;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}

export interface GameCardProps {
  game: ObjGame;
  onClick?: () => void;
  
  // Play actions
  onPlay: () => void;
  playLabel: string;
  
  // Continue session (optional)
  hasSession?: boolean;
  onContinue?: () => void;
  continueLabel?: string;
  onRestart?: () => void;
  restartLabel?: string;
  
  // Visibility badge (optional, for owner view)
  showVisibility?: boolean;
  
  // Creator info (optional, for AllGames view)
  showCreator?: boolean;
  isOwner?: boolean;
  creatorLabel?: string;
  
  // Favorite
  isFavorite?: boolean;
  onToggleFavorite?: () => void;
  favoriteLabel?: string;
  unfavoriteLabel?: string;
  
  // Action buttons (configurable)
  actions?: GameCardAction[];
  
  // Date display (optional)
  dateLabel?: string;
}

export function GameCard({
  game,
  onClick,
  onPlay,
  playLabel,
  hasSession = false,
  onContinue,
  continueLabel,
  onRestart,
  restartLabel,
  showVisibility = false,
  showCreator = false,
  isOwner = false,
  creatorLabel,
  isFavorite = false,
  onToggleFavorite,
  favoriteLabel,
  unfavoriteLabel,
  actions = [],
  dateLabel,
}: GameCardProps) {
  const { t } = useTranslation('common');

  const renderPlayButtons = () => {
    if (hasSession && onContinue) {
      return (
        <Group gap="xs" wrap="nowrap">
          <PlayGameButton onClick={onContinue} size="xs">
            {continueLabel}
          </PlayGameButton>
          {onRestart && (
            <TextButton onClick={onRestart} size="xs">
              {restartLabel}
            </TextButton>
          )}
        </Group>
      );
    }
    
    return (
      <PlayGameButton onClick={onPlay} size="xs">
        {playLabel}
      </PlayGameButton>
    );
  };

  const renderTopRightBadge = () => {
    if (showVisibility) {
      return game.public ? (
        <Badge size="xs" color="green" variant="light" leftSection={<IconWorld size={10} />} style={{ whiteSpace: 'nowrap', flexShrink: 0 }}>
          {t('games.visibility.public')}
        </Badge>
      ) : (
        <Badge size="xs" color="gray" variant="light" leftSection={<IconLock size={10} />} style={{ whiteSpace: 'nowrap', flexShrink: 0 }}>
          {t('games.visibility.private')}
        </Badge>
      );
    }
    
    if (showCreator) {
      if (isOwner) {
        return (
          <Badge size="xs" color="violet" variant="light">
            {creatorLabel || t('games.fields.me')}
          </Badge>
        );
      }
      if (game.creatorName) {
        return (
          <Text size="xs" c="gray.5">
            {game.creatorName}
          </Text>
        );
      }
    }
    
    return null;
  };

  return (
    <Card 
      shadow="sm" 
      p="md" 
      radius="md" 
      withBorder
      style={{ cursor: onClick ? 'pointer' : 'default' }}
      onClick={onClick}
    >
      <Stack gap="sm">
        {/* Top row: Title (left) + Date + Badge (right) */}
        <Group justify="space-between" align="flex-start" wrap="nowrap">
          <Text fw={600} size="md" lineClamp={1} style={{ flex: 1, minWidth: 0 }}>
            {game.name}
          </Text>
          <Group gap="xs" wrap="nowrap" style={{ flexShrink: 0 }}>
            {dateLabel && (
              <Text size="xs" c="gray.5" style={{ whiteSpace: 'nowrap' }}>
                {dateLabel}
              </Text>
            )}
            {renderTopRightBadge()}
          </Group>
        </Group>
        
        {/* Description */}
        {game.description && (
          <Text size="sm" c="gray.6" lineClamp={2}>
            {game.description}
          </Text>
        )}
        
        {/* Bottom row: Play buttons (left) + Actions (right) */}
        <Group justify="space-between" align="flex-end" wrap="nowrap">
          {/* Play/Continue/Restart buttons */}
          <Box onClick={(e) => e.stopPropagation()}>
            {renderPlayButtons()}
          </Box>
          
          {/* Action buttons + Favorite */}
          <Group gap={4} onClick={(e) => e.stopPropagation()} style={{ flexShrink: 0 }}>
            {actions.map((action) => (
              <Tooltip key={action.key} label={action.label} withArrow>
                {action.key === 'edit' ? (
                  <EditIconButton onClick={action.onClick} aria-label={action.label} />
                ) : action.key === 'delete' ? (
                  <DeleteIconButton onClick={action.onClick} aria-label={action.label} />
                ) : (
                  <GenericIconButton
                    icon={action.icon}
                    onClick={action.onClick}
                    aria-label={action.label}
                  />
                )}
              </Tooltip>
            ))}
            {onToggleFavorite && (
              <Tooltip label={isFavorite ? unfavoriteLabel : favoriteLabel} withArrow>
                <GenericIconButton
                  icon={isFavorite ? <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" /> : <IconStar size={18} />}
                  onClick={onToggleFavorite}
                  aria-label={(isFavorite ? unfavoriteLabel : favoriteLabel) || 'Toggle favorite'}
                />
              </Tooltip>
            )}
          </Group>
        </Group>
      </Stack>
    </Card>
  );
}
