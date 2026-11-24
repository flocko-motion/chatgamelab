import { Badge, Card, Tooltip, VisuallyHidden } from "@mantine/core";
import {
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
import { GenreChip } from "@components/GenreChip";
import { Critter } from "@components/Critter";
import { ItemTitle } from "@components/typography";
import classes from "./GameCard.module.css";

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
    | "creatorId"
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
 * asks whether to continue or start anew. The cover carries its genre, a
 * Private tag for a private game (public goes unsaid), Sponsored, and the
 * favourite star; below it the facts (plays, date), the name, who made it
 * (their critter) and the description; at the foot what a click does and
 * Edit (opens it in the editor), which line up across a row. Hover lifts
 * the card with a little wobble.
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

  return (
    <Card
      className={classes.card}
      data-sponsored={sponsored || undefined}
      padding={0}
      onClick={play}
      data-unplayable={held || undefined}
    >
      <div className={classes.cover}>
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
            <IconDeviceGamepad2 className={classes.playIcon} stroke={1.75} aria-hidden />
          </span>
        </button>
        <div className={classes.overlayTop}>
          {/* Public is the default and goes unsaid; a private game is marked. */}
          {showVisibility && !game.public && (
            <span className={classes.privateTag}>
              <IconLock size={12} aria-hidden />
              {t("games.visibility.private")}
            </span>
          )}
          {onToggleFavorite && (
            <div className={classes.star} onClick={(e) => e.stopPropagation()}>
              <Tooltip label={favoriteText} withArrow>
                <GenericIconButton
                  icon={isFavorite ? <IconStarFilled size={18} /> : <IconStar size={18} />}
                  color={isFavorite ? "orange" : "ink"}
                  onClick={onToggleFavorite}
                  aria-label={favoriteText ?? ""}
                />
              </Tooltip>
            </div>
          )}
        </div>
        <div className={classes.overlayBottom}>
          <GenreChip genre={game.genre} />
          {sponsored && (
            <Tooltip label={t("games.sponsor.sponsoredTooltip")} withArrow multiline w={250}>
              <span className={classes.sponsorTag} style={{ cursor: "help" }}>
                <IconHeartFilled size={12} aria-hidden />
                {t("games.sponsor.sponsored")}
              </span>
            </Tooltip>
          )}
        </div>
      </div>

      <div className={classes.body}>
        {/* The facts: how often played, when changed. */}
        <span className={classes.facts}>
          <IconPlayerPlay size={12} aria-hidden />
          <VisuallyHidden>{t("games.fields.playCount")}</VisuallyHidden>
          {game.playCount}
          {dateLabel && <span aria-hidden> · </span>}
          {dateLabel}
        </span>

        <ItemTitle>{game.name}</ItemTitle>

        {/* Who made it, always: games of the same name differ by their maker;
            one's own is marked, for whoever does not know their own name. */}
        {game.creatorName && (
          <span className={classes.creator}>
            <Critter seed={game.creatorId ?? game.creatorName} size={20} />
            <span className={classes.creatorName}>{game.creatorName}</span>
            {isOwner && <span className={classes.mine}>{t("games.badges.myGame")}</span>}
          </span>
        )}

        {game.description && <p className={classes.description}>{game.description}</p>}

        {(isWorkshopGame || extra) && (
          <div className={classes.tags}>
            {isWorkshopGame && (
              <Badge size="sm" color="accent" variant="light" leftSection={<IconSchool size={10} />}>
                {t("games.badges.workshop")}
              </Badge>
            )}
            {extra}
          </div>
        )}

        {problem && <UnplayableNote problem={problem} />}
      </div>

      {/* The foot: what a click does (left), editing (right). */}
      <div className={classes.foot} onClick={(e) => e.stopPropagation()}>
        {play && (
          <span className={classes.playText} aria-hidden>
            {playText} →
          </span>
        )}
        <EditButtonWithText onClick={onEdit}>{editLabel}</EditButtonWithText>
      </div>
    </Card>
  );
}
