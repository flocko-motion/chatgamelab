import { useNavigate } from "@tanstack/react-router";
import {
  useCurrentUser,
  useGamePage,
  useGameSessionMap,
  useOrganisation,
  useUserSessions,
  useUserStats,
  useWorkshopList,
} from "@/api/hooks";
import type { GameView, SessionView } from "@/api/types";
import { ROUTES } from "@/common/routes/routes";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { RECENT_WORKSHOPS_PRESET } from "../lib/recentWorkshops";

/** How many of each the dashboard shows. */
const COUNTS = { continue: 3, own: 4, fresh: 4, popular: 8, workshops: 4 } as const;

/**
 * Everything the dashboard shows, and where its parts lead: the user and
 * their numbers, the games in progress, their own games, new and popular
 * games of others, their workshops. The server lists, filters and counts.
 */
export function useDashboard() {
  const navigate = useNavigate();
  const { enterWorkshopMode } = useWorkshopMode();

  const { data: user } = useCurrentUser();
  const { data: stats } = useUserStats();
  const { data: sessions = [], isLoading: sessionsLoading } = useUserSessions();
  const { sessionMap } = useGameSessionMap();

  const institutionId = user?.role?.institution?.id;
  const { data: organisation } = useOrganisation(institutionId);
  const { items: workshops } = useWorkshopList(institutionId, RECENT_WORKSHOPS_PRESET);

  const own = useGamePage({ sortBy: "modifiedAt", sortDir: "desc", filters: ["own"], limit: COUNTS.own });
  const fresh = useGamePage({ sortBy: "createdAt", sortDir: "desc", filters: ["public", "others"], limit: COUNTS.fresh });
  const popular = useGamePage({ sortBy: "playCount", sortDir: "desc", filters: ["public"], limit: COUNTS.popular });
  // Only the count matters: how many public games the platform has.
  const publicGames = useGamePage({ filters: ["public"], limit: 1 });

  /** A game of someone else: continue the caller's session of it, or start one. */
  const play = (game: Pick<GameView, "id">) => {
    const session = sessionMap.get(game.id);
    navigate({ to: (session?.id ? `/sessions/${session.id}` : `/games/${game.id}/play`) as "/" });
  };

  return {
    user,
    /** The name to greet with: the first word of the user's name. */
    firstName: (user?.name ?? "").split(/[\s@]/)[0],
    numbers: {
      gamesOnPlatform: publicGames.data?.total,
      gamesCreated: stats?.gamesCreated,
      playsOfOwnGames: stats?.totalPlaysOnGames,
    },
    sessions: sessions.slice(0, COUNTS.continue),
    sessionsLoading,
    ownGames: own.data?.items ?? [],
    ownGamesLoading: own.isLoading,
    freshGames: fresh.data?.items ?? [],
    popularGames: popular.data?.items ?? [],
    workshops: institutionId ? workshops.slice(0, COUNTS.workshops) : [],
    mayCreateWorkshop: !!organisation?.mayCreateWorkshop,
    go: {
      createAbout: (about: string) =>
        navigate({ to: ROUTES.GAME_CREATE as "/", search: about.trim() ? { about: about.trim() } : {} }),
      session: (session: SessionView) => navigate({ to: `/sessions/${session.id}` as "/" }),
      play,
      open: (game: Pick<GameView, "id">) => navigate({ to: "/games/$gameId", params: { gameId: game.id } }),
      gallery: () => navigate({ to: ROUTES.ALL_GAMES as "/" }),
      ownGames: () => navigate({ to: ROUTES.ALL_GAMES as "/", search: { only: "own" } }),
      workshop: async (id: string, name: string) => {
        await enterWorkshopMode(id, name);
        navigate({ to: ROUTES.MY_WORKSHOP as "/" });
      },
      workshops: () =>
        navigate({ to: ROUTES.MY_ORGANIZATION as "/my-organization", search: { tab: "workshops" } }),
    },
  };
}

export type DashboardData = ReturnType<typeof useDashboard>;
