import { useMantineTheme } from "@mantine/core";
import { IconCompass, IconDeviceGamepad2, IconMessages, IconRobot, IconUsersGroup } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import classes from "./GenreChip.module.css";

const ICONS: Record<string, typeof IconCompass> = {
  adventure: IconCompass,
  npc: IconMessages,
  discussion: IconUsersGroup,
  chatbot: IconRobot,
};

export interface GenreChipProps {
  /** The game's genre key (e.g. "adventure", "npc"). */
  genre: string;
  size?: "sm" | "md";
}

/**
 * GenreChip - what kind of game it is, impossible to miss
 *
 * USE WHEN:
 * - Showing a game: on its cover, next to its name in a list
 *
 * Each genre has its own colour (theme.other.genreColors) and icon; the text
 * is ink on every fill (≥ 7.8:1).
 *
 * @example
 * <GenreChip genre={game.genre} />
 */
export function GenreChip({ genre, size = "md" }: GenreChipProps) {
  const { t } = useTranslation("common");
  const theme = useMantineTheme();
  const Icon = ICONS[genre] ?? IconDeviceGamepad2;
  const fill = theme.other.genreColors[genre] ?? theme.other.genreFallbackColor;
  return (
    <span className={classes.chip} data-size={size} style={{ background: fill }}>
      <Icon size={size === "sm" ? 13 : 15} stroke={2.2} aria-hidden />
      {t(`games.genres.${genre}`, { defaultValue: genre })}
    </span>
  );
}
