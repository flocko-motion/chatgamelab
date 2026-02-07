import { Card, Group, Badge, Stack, Text, Box, Tooltip } from "@mantine/core";
import {
  IconWorld,
  IconLock,
  IconStar,
  IconStarFilled,
  IconSchool,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import {
  PlayGameButton,
  TextButton,
  EditIconButton,
  DeleteIconButton,
  GenericIconButton,
} from "@components/buttons";
import type { ObjGame } from "@/api/generated";
import type { ReactNode } from "react";

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

  // Workshop indicator (for workshop mode)
  isWorkshopGame?: boolean;
}

/** Badge type for rendering multiple badges */
type GameBadgeType = "owner" | "workshop" | "public";

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
  isWorkshopGame = false,
}: GameCardProps) {
  const { t } = useTranslation("common");

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

  // Determine which badges to show
  const getBadges = (): GameBadgeType[] => {
    const badges: GameBadgeType[] = [];
    if (isOwner) badges.push("owner");
    if (isWorkshopGame) badges.push("workshop");
    if (game.public) badges.push("public");
    return badges;
  };

  const renderBadge = (type: GameBadgeType) => {
    switch (type) {
      case "owner":
        return (
          <Badge
            key="owner"
            size="xs"
            color="violet"
            variant="light"
            style={{ flexShrink: 0 }}
          >
            {creatorLabel || t("games.badges.myGame")}
          </Badge>
        );
      case "workshop":
        return (
          <Badge
            key="workshop"
            size="xs"
            color="cyan"
            variant="light"
            leftSection={<IconSchool size={10} />}
            style={{ flexShrink: 0 }}
          >
            {t("games.badges.workshop")}
          </Badge>
        );
      case "public":
        return (
          <Badge
            key="public"
            size="xs"
            color="green"
            variant="light"
            leftSection={<IconWorld size={10} />}
            style={{ flexShrink: 0 }}
          >
            {t("games.badges.public")}
          </Badge>
        );
      default:
        return null;
    }
  };

  const renderBadges = () => {
    const badges = getBadges();
    if (badges.length === 0) {
      // Show creator name if not owner and showCreator is enabled
      if (showCreator && game.creatorName) {
        return (
          <Text size="xs" c="gray.5">
            {game.creatorName}
          </Text>
        );
      }
      return null;
    }
    return (
      <Group gap={4} wrap="wrap">
        {badges.map(renderBadge)}
      </Group>
    );
  };

  // For showVisibility mode (e.g., MyGames page), show public/private badge
  const renderVisibilityBadge = () => {
    if (!showVisibility) return null;
    return game.public ? (
      <Badge
        size="xs"
        color="green"
        variant="light"
        leftSection={<IconWorld size={10} />}
        style={{ whiteSpace: "nowrap", flexShrink: 0 }}
      >
        {t("games.visibility.public")}
      </Badge>
    ) : (
      <Badge
        size="xs"
        color="gray"
        variant="light"
        leftSection={<IconLock size={10} />}
        style={{ whiteSpace: "nowrap", flexShrink: 0 }}
      >
        {t("games.visibility.private")}
      </Badge>
    );
  };

  return (
    <Card
      shadow="sm"
      p="md"
      radius="md"
      withBorder
      style={{ cursor: onClick ? "pointer" : "default" }}
      onClick={onClick}
    >
      <Stack gap="sm">
        {/* Top row: Title (left) + Date + Visibility (right) */}
        <Group justify="space-between" align="flex-start" wrap="nowrap">
          <Text
            fw={600}
            size="md"
            lineClamp={1}
            style={{ flex: 1, minWidth: 0 }}
          >
            {game.name}
          </Text>
          <Group gap="xs" wrap="nowrap" style={{ flexShrink: 0 }}>
            {dateLabel && (
              <Text size="xs" c="gray.5" style={{ whiteSpace: "nowrap" }}>
                {dateLabel}
              </Text>
            )}
            {renderVisibilityBadge()}
          </Group>
        </Group>

        {/* Badges row: Owner, Workshop, Public badges */}
        {(showCreator || isWorkshopGame || game.public || isOwner) && (
          <Box>{renderBadges()}</Box>
        )}

        {/* Description */}
        {game.description && (
          <Text size="sm" c="gray.6" lineClamp={2}>
            {game.description}
          </Text>
        )}

        {/* Bottom row: Play buttons (left) + Actions (right) */}
        <Group justify="space-between" align="flex-end" wrap="wrap">
          {/* Play/Continue/Restart buttons */}
          <Box onClick={(e) => e.stopPropagation()}>{renderPlayButtons()}</Box>

          {/* Action buttons + Favorite */}
          <Group
            gap={4}
            wrap="wrap"
            onClick={(e) => e.stopPropagation()}
          >
            {actions.map((action) => (
              <Tooltip key={action.key} label={action.label} withArrow>
                {action.key === "edit" ? (
                  <EditIconButton
                    onClick={action.onClick}
                    aria-label={action.label}
                  />
                ) : action.key === "delete" ? (
                  <DeleteIconButton
                    onClick={action.onClick}
                    aria-label={action.label}
                  />
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
              <Tooltip
                label={isFavorite ? unfavoriteLabel : favoriteLabel}
                withArrow
              >
                <GenericIconButton
                  icon={
                    isFavorite ? (
                      <IconStarFilled
                        size={18}
                        color="var(--mantine-color-yellow-5)"
                      />
                    ) : (
                      <IconStar size={18} />
                    )
                  }
                  onClick={onToggleFavorite}
                  aria-label={
                    (isFavorite ? unfavoriteLabel : favoriteLabel) ||
                    "Toggle favorite"
                  }
                />
              </Tooltip>
            )}
          </Group>
        </Group>
      </Stack>
    </Card>
  );
}
