import { Badge, Card, Tooltip, VisuallyHidden, useMantineTheme } from "@mantine/core";
import {
  IconLock,
  IconStar,
  IconStarFilled,
  IconSchool,
  IconHeartFilled,
  IconPlayerPlay,
  IconPlayerPlayFilled,
  IconEdit,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { blocksContinue, unplayable } from "@/common/lib/playability";
import { UnplayableNote } from "@components/UnplayableNote";
import { GenericIconButton } from "@components/buttons";
import type { GameView } from "@/api/types";
import type { CSSProperties, ReactNode } from "react";
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

  /**
   * The smaller tile, for a few games among other things (the dashboard):
   * the same parts and actions, tighter: no description, no "my game"
   * badge.
   */
  compact?: boolean;
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
 * the card with a little wobble. Compact, it is the same tile, smaller.
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
  compact = false,
}: GameCardProps) {
  const { t } = useTranslation("common");
  const theme = useMantineTheme();
  // On hover the shadow takes the genre's colour, as its chip on the cover.
  const genreColor: string | undefined = theme.other.genreColors[game.genre];
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
      style={genreColor ? ({ "--genre-color": genreColor } as CSSProperties) : undefined}
      data-sponsored={sponsored || undefined}
      padding={0}
      onClick={play}
      data-unplayable={held || undefined}
      data-compact={compact || undefined}
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
            <IconPlayerPlayFilled className={classes.playIcon} aria-hidden />
            {/* Shown instead while the pointer is on Edit: what that click does. */}
            <IconEdit className={classes.editIcon} aria-hidden />
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
          <GenreChip genre={game.genre} size={compact ? "sm" : undefined} />
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
            {isOwner && !compact && <span className={classes.mine}>{t("games.badges.myGame")}</span>}
          </span>
        )}

        {!compact && game.description && <p className={classes.description}>{game.description}</p>}

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

      {/* The foot: what a click does (left), editing (right). The play text
          is part of the tile, so a click on it plays; Edit only edits. */}
      <div className={classes.foot}>
        {play && (
          <span className={classes.playText} aria-hidden>
            {playText}
          </span>
        )}
        <span className={classes.edit} onClick={(e) => e.stopPropagation()}>
          {/* Plain text, as Play is; the pencil shows on the cover. */}
          <button
            type="button"
            className={classes.editText}
            onClick={onEdit}
            aria-label={`${editLabel}: ${game.name}`}
          >
            {editLabel}
          </button>
        </span>
      </div>
    </Card>
  );
}
