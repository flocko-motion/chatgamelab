import { useState } from "react";
import { IconSparkles } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { gameCoverUrl, useCoverDrawing } from "@/api/hooks";
import { coverTile } from "@/common/lib/coverTile";
import type { GameView } from "@/api/types";
import classes from "./GameCover.module.css";

export interface GameCoverProps {
  game: Pick<GameView, "id" | "name" | "coverAt">;
  /** A cover is being drawn: a spinner and a note over what shows now. It
   * also shows by itself while this browser draws the game's cover. */
  busy?: boolean;
  /** Edge to edge, square corners: inside a card section that rounds them. */
  flush?: boolean;
  /** The frame's shape: 3:2 (default), square, or filling its container. */
  ratio?: "3/2" | "1/1" | "fill";
}

/**
 * GameCover - a game's cover picture, in a 3:2 frame
 *
 * USE WHEN:
 * - Showing what a game looks like: in the editor, on a game's tile; it
 *   fills the width its container gives it
 *
 * A game without a cover shows a tile with the first letter of its name, on
 * a gradient of theme colours that its id picks (coverTile): the same game
 * always looks the same, and a list of games looks complete either way.
 *
 * @example
 * <GameCover game={game} />
 */
export function GameCover({ game, busy = false, flush = false, ratio = "3/2" }: GameCoverProps) {
  const { t } = useTranslation("common");
  const url = gameCoverUrl(game);
  const drawing = useCoverDrawing(game.id) || busy;
  // A picture that would not load shows the tile, never a broken image.
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const initial = Array.from(game.name.trim())[0]?.toLocaleUpperCase() ?? "";

  return (
    <div className={flush ? `${classes.frame} ${classes.flush}` : classes.frame} data-ratio={ratio}>
      {url && url !== failedUrl ? (
        <img
          className={classes.image}
          src={url}
          alt={t("games.cover.alt", { name: game.name })}
          loading="lazy"
          decoding="async"
          onError={() => setFailedUrl(url)}
        />
      ) : (
        <div
          className={classes.fallback}
          style={{ background: coverTile(game.id).background }}
          role="img"
          aria-label={t("games.cover.none")}
        >
          <svg className={classes.fallbackInitial} viewBox="0 0 100 100" aria-hidden>
            <text x="50" y="50" textAnchor="middle" dominantBaseline="central">
              {initial}
            </text>
          </svg>
        </div>
      )}
      {drawing && (
        <div className={classes.busy} role="status">
          <span className={classes.ring}>
            <IconSparkles className={classes.sparkle} stroke={2} aria-hidden />
          </span>
          <span className={classes.busyText}>{t("games.cover.drawing")}</span>
        </div>
      )}
    </div>
  );
}
