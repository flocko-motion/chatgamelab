import { IconCompass, IconDeviceGamepad2, IconMessages, IconRobot, IconUsersGroup } from "@tabler/icons-react";

/** Each genre's icon, as its chip shows it. */
export const GENRE_ICONS: Record<string, typeof IconCompass> = {
  adventure: IconCompass,
  npc: IconMessages,
  discussion: IconUsersGroup,
  chatbot: IconRobot,
};

/** The icon of a genre without one of its own. */
export const GENRE_FALLBACK_ICON = IconDeviceGamepad2;
