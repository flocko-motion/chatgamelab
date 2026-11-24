import { useNavigate, useRouterState } from "@tanstack/react-router";
import { ROUTES } from "@/common/routes/routes";

export type MobileTabKey = "home" | "games" | "create" | "me";

export interface MobileTab {
  key: MobileTabKey;
  active: boolean;
  open: () => void;
}

/** The pages the Me tab stands for: the user's own. */
const OWN_PAGES: readonly string[] = [ROUTES.PROFILE, ROUTES.SETTINGS, ROUTES.API_KEYS];

/**
 * The phone's tab bar: Dashboard, Games, Create and Me (the profile), each
 * with whether it is the current page and how to go there.
 */
export function useMobileTabs(): MobileTab[] {
  const navigate = useNavigate();
  const path = useRouterState({ select: (s) => s.location.pathname });
  const go = (to: string) => () => navigate({ to: to as "/" });
  const isGames = path === ROUTES.ALL_GAMES || path === `${ROUTES.ALL_GAMES}/`;
  return [
    { key: "home", active: path.startsWith(ROUTES.DASHBOARD), open: go(ROUTES.DASHBOARD) },
    { key: "games", active: isGames, open: go(ROUTES.ALL_GAMES) },
    { key: "create", active: path.startsWith(ROUTES.GAME_CREATE), open: go(ROUTES.GAME_CREATE) },
    { key: "me", active: OWN_PAGES.some((p) => path.startsWith(p)), open: go(ROUTES.PROFILE) },
  ];
}
