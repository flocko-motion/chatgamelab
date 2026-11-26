import type { ReactNode } from "react";
import { useMantineTheme } from "@mantine/core";
import { IconBuilding, IconStar, IconUser, IconWorld } from "@tabler/icons-react";
import { GENRE_FALLBACK_ICON, GENRE_ICONS } from "@components/GenreChip";

const ICONS: Record<string, ReactNode> = {
  favorites: <IconStar size={14} aria-hidden />,
  own: <IconUser size={14} aria-hidden />,
  organisation: <IconBuilding size={14} aria-hidden />,
  public: <IconWorld size={14} aria-hidden />,
};

/**
 * How the game list's filters look in the filter bar: an icon for who/where
 * (favorites, mine, organisation, public); for a genre the icon of its chip,
 * and its colour, which the filter takes while on.
 */
export function useGameFilterLook() {
  const theme = useMantineTheme();
  return (value: string): { icon?: ReactNode; color?: string } | undefined => {
    if (value.startsWith("genre:")) {
      const genre = value.slice("genre:".length);
      const Icon = GENRE_ICONS[genre] ?? GENRE_FALLBACK_ICON;
      return {
        icon: <Icon size={14} aria-hidden />,
        color: theme.other.genreColors[genre] ?? theme.other.genreFallbackColor,
      };
    }
    return ICONS[value] ? { icon: ICONS[value] } : undefined;
  };
}
