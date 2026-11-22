import { useState } from "react";
import { Loader } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { gameCoverUrl } from "@/api/hooks";
import { coverTile } from "@/common/lib/coverTile";
import type { GameView } from "@/api/types";
import classes from "./GameCover.module.css";

export interface GameCoverProps {
  game: Pick<GameView, "id" | "name" | "coverAt">;
  /** A cover is being drawn: a spinner over what shows now. */
  busy?: boolean;
  /** Edge to edge, square corners: inside a card section that rounds them. */
  flush?: boolean;
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
export function GameCover({ game, busy = false, flush = false }: GameCoverProps) {
  const { t } = useTranslation("common");
  const url = gameCoverUrl(game);
  // A picture that would not load shows the tile, never a broken image.
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const initial = Array.from(game.name.trim())[0]?.toLocaleUpperCase() ?? "";

  return (
    <div className={flush ? `${classes.frame} ${classes.flush}` : classes.frame}>
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
      {busy && (
        <div className={classes.busy}>
          <Loader color="white" />
        </div>
      )}
    </div>
  );
}
