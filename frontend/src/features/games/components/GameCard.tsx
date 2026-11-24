import {
  Card,
  Group,
  Badge,
  Stack,
  Text,
  Tooltip,
  VisuallyHidden,
} from "@mantine/core";
import {
  IconWorld,
  IconLock,
  IconStar,
  IconStarFilled,
  IconSchool,
  IconHeartFilled,
  IconPlayerPlay,
  IconDeviceGamepad2,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { blocksContinue, unplayable } from "@/common/lib/playability";
import { UnplayableNote } from "@components/UnplayableNote";
import {
  EditButtonWithText,
  GenericIconButton,
} from "@components/buttons";
import type { GameView } from "@/api/types";
import type { ReactNode } from "react";
import { GameCover } from "@components/GameCover";
import { GameActionIcon } from "./GameListCells";
import classes from "./GameCard.module.css";

export interface GameCardAction {
  key: string;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}

export interface GameCardProps {
  game: Pick<
    GameView,
    | "id"
    | "name"
    | "genre"
    | "coverAt"
    | "description"
    | "public"
    | "publicSponsoredApiKeyShareId"
    | "creatorName"
    | "playCount"
    | "playability"
  >;
  /** Opens the game: to edit it, or to look at it for who may not. */
  onEdit: () => void;
  editLabel: string;

  /**
   * A click on the tile opens the game: it starts, or, with a session of it,
   * asks whether to continue or start anew. `canStartNew` false: only
   * continuing is possible.
   */
  onOpen: (canStartNew: boolean) => void;
  playLabel: string;

  // The caller's session of it (optional)
  hasSession?: boolean;
  continueLabel?: string;

  /** Public or private on the cover; otherwise only public games say so. */
  showVisibility?: boolean;

  // Owner badge
  isOwner?: boolean;

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

  // Extra content among the facts (e.g. the workshop, for admins)
  extra?: ReactNode;
}

/**
 * GameCard - a game as a tile in a game list
 *
 * A click on the tile opens the game: it starts, or, with a session of it,
 * asks whether to continue or start anew. The cover on top carries where the game is seen (public, private,
 * sponsored) and the favourite star; below it the name, the facts (who
 * made it, how often it was played, when it changed) and the description;
 * at the foot the button that opens the game and the actions, which line up
 * across a row.
 */
export function GameCard({
  game,
  onEdit,
  editLabel,
  onOpen,
  playLabel,
  hasSession = false,
  continueLabel,
  showVisibility = false,
  isOwner = false,
  isFavorite = false,
  onToggleFavorite,
  favoriteLabel,
  unfavoriteLabel,
  actions = [],
  dateLabel,
  isWorkshopGame = false,
  extra,
}: GameCardProps) {
  const { t } = useTranslation("common");
  const sponsored = !!game.publicSponsoredApiKeyShareId;
  const favoriteText = isFavorite ? unfavoriteLabel : favoriteLabel;
  const continues = hasSession;
  // A game the caller cannot start says why, and a click on it plays nothing.
  const problem = unplayable(game);
  const held = problem !== null && (!continues || blocksContinue(problem));
  const play = held ? undefined : () => onOpen(problem === null);
  const playText = continues ? continueLabel : playLabel;

  const visibility = game.public ? (
    <Badge
      className={classes.overlayBadge}
      size="sm"
      color="green"
      variant="white"
      leftSection={<IconWorld size={12} />}
    >
      {t("games.visibility.public")}
    </Badge>
  ) : showVisibility ? (
    <Badge
      className={classes.overlayBadge}
      size="sm"
      color="gray"
      variant="white"
      leftSection={<IconLock size={12} />}
    >
      {t("games.visibility.private")}
    </Badge>
  ) : null;

  return (
    <Card
      className={classes.card}
      data-sponsored={sponsored || undefined}
      shadow="sm"
      padding="md"
      radius="lg"
      withBorder
      onClick={play}
      data-unplayable={held || undefined}
    >
      <Card.Section className={classes.cover}>
        <GameCover game={game} flush />
        {/* What a click does, shown on hover; the keyboard's way in. */}
        <button
          type="button"
          className={classes.playHint}
          onClick={(e) => {
            e.stopPropagation();
            play?.();
          }}
          disabled={!play}
          aria-label={`${playText}: ${game.name}`}
        >
          <span className={classes.playCircle}>
            <IconDeviceGamepad2
              className={classes.playIcon}
              stroke={1.75}
              aria-hidden
            />
          </span>
        </button>
        <div className={classes.overlayTop}>
          {visibility}
          {onToggleFavorite && (
            <div className={classes.star} onClick={(e) => e.stopPropagation()}>
              <Tooltip label={favoriteText} withArrow>
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
                  aria-label={favoriteText ?? ""}
                />
              </Tooltip>
            </div>
          )}
        </div>
        {sponsored && (
          <div className={classes.overlayBottom}>
            <Tooltip
              label={t("games.sponsor.sponsoredTooltip")}
              withArrow
              multiline
              w={250}
            >
              <Badge
                className={classes.overlayBadge}
                size="sm"
                color="pink"
                variant="filled"
                leftSection={<IconHeartFilled size={12} />}
                style={{ cursor: "help" }}
              >
                {t("games.sponsor.sponsored")}
              </Badge>
            </Tooltip>
          </div>
        )}
      </Card.Section>

      <Stack gap={6} mt="sm" className={classes.body}>
        <Text fw={700} size="md" lineClamp={2}>
          {game.name}
        </Text>

        {/* Who made it, always: games of the same name differ by their maker;
            one's own is marked, for whoever does not know their own name. */}
        {game.creatorName && (
          <Group gap={6} wrap="nowrap">
            <Text size="sm" fw={500} lineClamp={1}>
              {game.creatorName}
            </Text>
            {isOwner && (
              <Badge size="xs" color="violet" variant="light" style={{ flexShrink: 0 }}>
                {t("games.badges.myGame")}
              </Badge>
            )}
          </Group>
        )}

        {/* The facts: how often played, when changed. */}
        <Group gap={6} wrap="wrap" className={classes.facts}>
          <Tooltip label={t("games.fields.playCount")} withArrow>
            <Group gap={2} wrap="nowrap" component="span">
              <IconPlayerPlay size={12} aria-hidden />
              <VisuallyHidden>{t("games.fields.playCount")}</VisuallyHidden>
              <span>{game.playCount}</span>
            </Group>
          </Tooltip>
          {dateLabel && <span aria-hidden>·</span>}
          {dateLabel && <span>{dateLabel}</span>}
        </Group>

        <Group gap={4} wrap="wrap">
          <Badge size="xs" color="gray" variant="light">
            {t(`games.genres.${game.genre}`, { defaultValue: game.genre })}
          </Badge>
          {isWorkshopGame && (
            <Badge
              size="xs"
              color="cyan"
              variant="light"
              leftSection={<IconSchool size={10} />}
            >
              {t("games.badges.workshop")}
            </Badge>
          )}
          {extra}
        </Group>

        {game.description && (
          <Text size="sm" c="gray.6" lineClamp={2}>
            {game.description}
          </Text>
        )}

        {problem && <UnplayableNote problem={problem} />}
      </Stack>

      {/* The foot: open the game (left), the actions (right). */}
      <Group
        className={classes.foot}
        justify="space-between"
        align="center"
        wrap="wrap"
        gap="xs"
        onClick={(e) => e.stopPropagation()}
      >
        <EditButtonWithText onClick={onEdit}>{editLabel}</EditButtonWithText>
        <Group gap={2} wrap="nowrap">
          {actions.map((action) => (
            <GameActionIcon key={action.key} action={action} />
          ))}
        </Group>
      </Group>
    </Card>
  );
}
