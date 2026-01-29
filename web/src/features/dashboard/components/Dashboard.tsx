import { useMemo } from "react";
import { Stack, SimpleGrid } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { ROUTES } from "@/common/routes/routes";
import { AppLayout, type NavItem } from "@/common/components/Layout";
import { navigationLogger } from "@/config/logger";
import { EXTERNAL_LINKS } from "@/config/externalLinks";
import { formatRelativeTime } from "@/common/lib/formatters";
import {
  useGames,
  useUserSessions,
  useGameSessionMap,
  useCurrentUser,
} from "@/api/hooks";
import {
  IconPlayerPlay,
  IconEdit,
  IconBuilding,
  IconUsers,
  IconSchool,
  IconTools,
  IconDeviceGamepad2,
} from "@tabler/icons-react";
import { InformationalCard, type ListItem } from "./InformationalCard";
import { QuickActionCard } from "./QuickActionCard";
import { LinksCard, type LinkItem } from "./LinkCard";

// My Games Card - shows last edited/created games
function MyGamesCard() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();
  const { data: games, isLoading } = useGames({
    sortBy: "modifiedAt",
    sortDir: "desc",
    filter: "own",
  });

  const recentGames = useMemo(() => {
    if (!games) return [];
    return games.slice(0, 5);
  }, [games]);

  const items: ListItem[] = recentGames.map((game) => ({
    id: game.id ?? "",
    label: game.name ?? t("untitled"),
    sublabel: formatRelativeTime(game.meta?.modifiedAt ?? game.meta?.createdAt),
    onClick: () => navigate({ to: `/my-games/${game.id}` as "/" }),
  }));

  return (
    <InformationalCard
      title={t("cards.myGames.title")}
      items={items}
      emptyMessage={t("cards.myGames.empty")}
      viewAllLabel={t("cards.myGames.viewAll")}
      onViewAll={() => navigate({ to: ROUTES.MY_GAMES as "/" })}
      isLoading={isLoading}
    />
  );
}

// My Rooms Card - placeholder for rooms
function MyRoomsCard() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();

  const items: ListItem[] = [];

  return (
    <InformationalCard
      title={t("cards.myRooms.title")}
      items={items}
      emptyMessage={t("cards.myRooms.empty")}
      viewAllLabel={t("cards.myRooms.viewAll")}
      onViewAll={() => navigate({ to: "/rooms" as "/" })}
      isLoading={false}
    />
  );
}

// Last Played Card - shows recent play sessions
function LastPlayedCard() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();
  const { data: sessions, isLoading } = useUserSessions();

  const recentSessions = useMemo(() => {
    if (!sessions) return [];
    return sessions.slice(0, 5);
  }, [sessions]);

  const items: ListItem[] = recentSessions.map((session) => ({
    id: session.id ?? "",
    label: session.gameName ?? t("untitledGame"),
    sublabel: formatRelativeTime(session.meta?.modifiedAt ?? session.meta?.createdAt),
    onClick: () => navigate({ to: `/sessions/${session.id}` as "/" }),
  }));

  return (
    <InformationalCard
      title={t("cards.lastPlayed.title")}
      items={items}
      emptyMessage={t("cards.lastPlayed.empty")}
      viewAllLabel={t("cards.lastPlayed.viewAll")}
      onViewAll={() => navigate({ to: ROUTES.ALL_GAMES as "/" })}
      isLoading={isLoading}
    />
  );
}

// Popular Games Card - shows top 10 most played games
function PopularGamesCard() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();
  const { data: games, isLoading: gamesLoading } = useGames({
    sortBy: "playCount",
    sortDir: "desc",
    filter: "public",
  });
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();

  const popularGames = useMemo(() => {
    if (!games) return [];
    return games.slice(0, 10);
  }, [games]);

  const handleGameClick = (gameId: string) => {
    const existingSession = sessionMap.get(gameId);
    if (existingSession?.id) {
      navigate({ to: `/sessions/${existingSession.id}` as "/" });
    } else {
      navigate({ to: `/play/${gameId}` as "/" });
    }
  };

  const items: ListItem[] = popularGames.map((game) => ({
    id: game.id ?? "",
    label: game.name ?? t("untitled"),
    sublabel: t("cards.popularGames.plays", { count: game.playCount ?? 0 }),
    onClick: () => handleGameClick(game.id ?? ""),
  }));

  return (
    <InformationalCard
      title={t("cards.popularGames.title")}
      items={items}
      emptyMessage={t("cards.popularGames.empty")}
      viewAllLabel={t("cards.popularGames.viewAll")}
      onViewAll={() => navigate({ to: ROUTES.ALL_GAMES as "/" })}
      isLoading={gamesLoading || sessionsLoading}
      maxItems={10}
    />
  );
}

// New Games Card - shows 10 newest games (excluding user's own)
function NewGamesCard() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();
  const { data: games, isLoading: gamesLoading } = useGames({
    sortBy: "createdAt",
    sortDir: "desc",
    filter: "public",
  });
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const { data: currentUser } = useCurrentUser();

  /* eslint-disable react-hooks/preserve-manual-memoization -- Optional chaining limitation */
  const newGames = useMemo(() => {
    if (!games || !currentUser?.id) return [];
    // Exclude user's own games
    return games
      .filter((game) => game.creatorId !== currentUser.id)
      .slice(0, 10);
  }, [games, currentUser?.id]);
  /* eslint-enable react-hooks/preserve-manual-memoization */

  const handleGameClick = (gameId: string) => {
    const existingSession = sessionMap.get(gameId);
    if (existingSession?.id) {
      navigate({ to: `/sessions/${existingSession.id}` as "/" });
    } else {
      navigate({ to: `/play/${gameId}` as "/" });
    }
  };

  const items: ListItem[] = newGames.map((game) => ({
    id: game.id ?? "",
    label: game.name ?? t("untitled"),
    sublabel: formatRelativeTime(game.meta?.createdAt),
    onClick: () => handleGameClick(game.id ?? ""),
  }));

  return (
    <InformationalCard
      title={t("cards.newGames.title")}
      items={items}
      emptyMessage={t("cards.newGames.empty")}
      viewAllLabel={t("cards.newGames.viewAll")}
      onViewAll={() => navigate({ to: ROUTES.ALL_GAMES as "/" })}
      isLoading={gamesLoading || sessionsLoading}
      maxItems={10}
    />
  );
}

// External Links Card
function ExternalLinksCard() {
  const { t } = useTranslation("dashboard");

  const links: LinkItem[] = [
    {
      id: EXTERNAL_LINKS.CHATGAMELAB.id,
      title: t("cards.externalLinks.mainSite.title"),
      description: t("cards.externalLinks.mainSite.description"),
      href: EXTERNAL_LINKS.CHATGAMELAB.href,
      icon: <IconSchool size={16} />,
    },
    {
      id: EXTERNAL_LINKS.JFF.id,
      title: t("cards.externalLinks.jff.title"),
      description: t("cards.externalLinks.jff.description"),
      href: EXTERNAL_LINKS.JFF.href,
      icon: <IconUsers size={16} />,
    },
  ];

  return (
    <LinksCard
      title={t("cards.externalLinks.title")}
      links={links}
      highlighted
    />
  );
}

// Quick Actions Card
function QuickActionsCard() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();

  const actions = [
    {
      id: "start-new-game",
      label: t("quickActions.startNewGame"),
      icon: <IconDeviceGamepad2 size={16} />,
      onClick: () => navigate({ to: ROUTES.ALL_GAMES as "/" }),
    },
    {
      id: "create-game",
      label: t("quickActions.createNewGame"),
      icon: <IconTools size={16} />,
      onClick: () => navigate({ to: (ROUTES.MY_GAMES + "/create") as "/" }),
    },
    {
      id: "create-room",
      label: t("quickActions.createRoom"),
      icon: <IconBuilding size={16} />,
      onClick: () => navigate({ to: "/rooms" as "/" }),
      disabled: true,
    },
    {
      id: "invite-members",
      label: t("quickActions.inviteMembers"),
      icon: <IconUsers size={16} />,
      onClick: () => {},
      disabled: true,
    },
  ];

  return <QuickActionCard title={t("quickActions.title")} actions={actions} />;
}

/**
 * Dashboard content component - can be used standalone or within AppLayout
 */
export function DashboardContent() {
  return (
    <Stack gap="xl">
      <SimpleGrid cols={{ base: 1, md: 2, lg: 3 }} spacing="lg">
        <MyGamesCard />
        <MyRoomsCard />
        <LastPlayedCard />
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="lg">
        <QuickActionsCard />
        <ExternalLinksCard />
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="lg">
        <PopularGamesCard />
        <NewGamesCard />
      </SimpleGrid>
    </Stack>
  );
}

/**
 * Full Dashboard page with layout - for standalone use
 */
export function Dashboard() {
  const { t } = useTranslation("navigation");

  const navItems: NavItem[] = [
    { label: t("play"), icon: <IconPlayerPlay size={18} />, onClick: () => {} },
    { label: t("create"), icon: <IconEdit size={18} />, onClick: () => {} },
    { label: t("rooms"), icon: <IconBuilding size={18} />, onClick: () => {} },
    { label: t("groups"), icon: <IconUsers size={18} />, onClick: () => {} },
  ];

  return (
    <AppLayout
      variant="authenticated"
      navItems={navItems}
      headerProps={{
        onSettingsClick: () => navigationLogger.debug("Settings clicked"),
        onProfileClick: () => navigationLogger.debug("Profile clicked"),
      }}
    >
      <DashboardContent />
    </AppLayout>
  );
}
