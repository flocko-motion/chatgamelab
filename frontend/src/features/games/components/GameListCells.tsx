import { Badge, Group, Stack, Text, Tooltip } from "@mantine/core";
import {
  IconHeartFilled,
  IconLock,
  IconSchool,
  IconUser,
  IconWorld,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { GameCover } from "@components/GameCover";

/* The cells of a game list's columns (useGameColumns). */

export function GameNameCell({
  game,
  isOwner,
}: {
  game: GameView;
  isOwner: boolean;
}) {
  const { t } = useTranslation("common");
  return (
    <Group gap="sm" wrap="nowrap" align="center">
      <div style={{ width: 72, flexShrink: 0 }}>
        <GameCover game={game} />
      </div>
      <Stack gap={2} style={{ minWidth: 0 }}>
        <Group gap="xs" wrap="nowrap">
          <Text fw={600} size="sm" c="gray.8" lineClamp={1}>
            {game.name}
          </Text>
          {isOwner && (
            <Badge
              size="xs"
              color="violet"
              variant="light"
              style={{ flexShrink: 0 }}
            >
              {t("games.badges.myGame")}
            </Badge>
          )}
          {game.publicSponsoredApiKeyShareId && (
            <Tooltip
              label={t("games.sponsor.sponsoredTooltip")}
              withArrow
              multiline
              w={250}
            >
              <Badge
                size="xs"
                color="pink"
                variant="light"
                leftSection={<IconHeartFilled size={10} />}
                style={{ flexShrink: 0, cursor: "help" }}
              >
                {t("games.sponsor.sponsored")}
              </Badge>
            </Tooltip>
          )}
        </Group>
        {game.description && (
          <Text size="xs" c="gray.5" lineClamp={1}>
            {game.description}
          </Text>
        )}
      </Stack>
    </Group>
  );
}

export function VisibilityBadge({ isPublic }: { isPublic: boolean }) {
  const { t } = useTranslation("common");
  return isPublic ? (
    <Badge
      size="sm"
      color="green"
      variant="light"
      leftSection={<IconWorld size={12} />}
      style={{ whiteSpace: "nowrap" }}
    >
      {t("games.visibility.public")}
    </Badge>
  ) : (
    <Badge
      size="sm"
      color="gray"
      variant="light"
      leftSection={<IconLock size={12} />}
      style={{ whiteSpace: "nowrap" }}
    >
      {t("games.visibility.private")}
    </Badge>
  );
}

/** Where a game in a workshop list comes from: the workshop, or public. */
export function GameBadges({
  game,
  isWorkshopGame,
}: {
  game: GameView;
  isWorkshopGame: boolean;
}) {
  const { t } = useTranslation("common");
  return (
    <Stack gap={4} align="flex-start">
      {isWorkshopGame && (
        <Badge
          size="xs"
          color="accent"
          variant="light"
          leftSection={<IconSchool size={10} />}
        >
          {t("games.badges.workshop")}
        </Badge>
      )}
      {game.public && (
        <Badge
          size="xs"
          color="green"
          variant="light"
          leftSection={<IconWorld size={10} />}
        >
          {t("games.visibility.public")}
        </Badge>
      )}
      {!isWorkshopGame && !game.public && (
        <Badge
          size="xs"
          color="gray"
          variant="light"
          leftSection={<IconUser size={10} />}
        >
          {t("games.visibility.private")}
        </Badge>
      )}
    </Stack>
  );
}
