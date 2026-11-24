import type { ReactNode } from "react";
import { useMantineTheme } from "@mantine/core";
import { IconBuilding, IconStar, IconUser, IconWorld } from "@tabler/icons-react";

const ICONS: Record<string, ReactNode> = {
  favorites: <IconStar size={14} aria-hidden />,
  own: <IconUser size={14} aria-hidden />,
  organisation: <IconBuilding size={14} aria-hidden />,
  public: <IconWorld size={14} aria-hidden />,
};

/**
 * How the game list's filters look in the filter bar: an icon for who/where
 * (favorites, mine, organisation, public), the genre's colour as a dot for a genre.
 */
export function useGameFilterLook() {
  const theme = useMantineTheme();
  return (value: string): { icon?: ReactNode; dot?: string } | undefined => {
    if (value.startsWith("genre:")) {
      const genre = value.slice("genre:".length);
      return { dot: theme.other.genreColors[genre] ?? theme.other.genreFallbackColor };
    }
    return ICONS[value] ? { icon: ICONS[value] } : undefined;
  };
}
