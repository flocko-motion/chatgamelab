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
import {
  EditButtonWithText,
  TextButton,
  GenericIconButton,
} from "@components/buttons";
import type { GameView } from "@/api/generated/api.gen";
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
  >;
  /** Opens the game: to edit it, or to look at it for who may not. */
  onEdit: () => void;
  editLabel: string;

  // Play actions: a click on the tile plays (or continues)
  onPlay: () => void;
  playLabel: string;

  // Continue session (optional)
  hasSession?: boolean;
  onContinue?: () => void;
  continueLabel?: string;
  onRestart?: () => void;
  restartLabel?: string;

  /** Public or private on the cover; otherwise only public games say so. */
  showVisibility?: boolean;

  // Owner badge
  isOwner?: boolean;
  // Who made the game, for games that aren't the caller's
  showCreator?: boolean;

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
 * A click on the tile plays the game, or continues the caller's session of
 * it. The cover on top carries where the game is seen (public, private,
 * sponsored) and the favourite star; below it the name, the facts (who
 * made it, how often it was played, when it changed) and the description;
 * at the foot the button that opens the game and the actions, which line up
 * across a row.
 */
export function GameCard({
  game,
  onEdit,
  editLabel,
  onPlay,
  playLabel,
  hasSession = false,
  onContinue,
  continueLabel,
  onRestart,
  restartLabel,
  showVisibility = false,
  isOwner = false,
  showCreator = false,
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
  const continues = hasSession && !!onContinue;
  const play = continues ? onContinue : onPlay;
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

  const creator = showCreator && !isOwner && game.creatorName;

  return (
    <Card
      className={classes.card}
      data-sponsored={sponsored || undefined}
      shadow="sm"
      padding="md"
      radius="lg"
      withBorder
      onClick={play}
    >
      <Card.Section className={classes.cover}>
        <GameCover game={game} flush />
        {/* What a click does, shown on hover; the keyboard's way in. */}
        <button
          type="button"
          className={classes.playHint}
          onClick={(e) => {
            e.stopPropagation();
            play();
          }}
          aria-label={`${playText}: ${game.name}`}
        >
          <span className={classes.playCircle}>
            <IconDeviceGamepad2 className={classes.playIcon} stroke={1.75} aria-hidden />
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

        {/* The facts: who made it, how often played, when changed. */}
        <Group gap={6} wrap="wrap" className={classes.facts}>
          {creator && (
            <span>{t("games.byCreator", { name: game.creatorName })}</span>
          )}
          {creator && <span aria-hidden>·</span>}
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
          {isOwner && (
            <Badge size="xs" color="violet" variant="light">
              {t("games.badges.myGame")}
            </Badge>
          )}
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
        <Group gap="xs" wrap="nowrap">
          <EditButtonWithText onClick={onEdit}>{editLabel}</EditButtonWithText>
          {continues && onRestart && (
            <TextButton onClick={onRestart} size="xs">
              {restartLabel}
            </TextButton>
          )}
        </Group>
        <Group gap={2} wrap="nowrap">
          {actions.map((action) => (
            <GameActionIcon key={action.key} action={action} />
          ))}
        </Group>
      </Group>
    </Card>
  );
}
