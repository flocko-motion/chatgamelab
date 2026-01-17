import { useMemo } from 'react';
import { 
  Stack, 
  SimpleGrid,
} from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { ROUTES } from '@/common/routes/routes';
import { AppLayout, type NavItem } from '@/common/components/Layout';
import { navigationLogger } from '@/config/logger';
import { EXTERNAL_LINKS } from '@/config/externalLinks';
import { useGames, useUserSessions } from '@/api/hooks';
import { 
  IconPlayerPlay, 
  IconEdit, 
  IconBuilding, 
  IconUsers, 
  IconSchool,
  IconTools,
  IconDeviceGamepad2,
} from '@tabler/icons-react';
import { InformationalCard, type ListItem } from './InformationalCard';
import { QuickActionCard } from './QuickActionCard';
import { LinksCard, type LinkItem } from './LinkCard';

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return '';
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

// My Games Card - shows last edited/created games
function MyGamesCard() {
  const { t } = useTranslation('dashboard');
  const navigate = useNavigate();
  const { data: games, isLoading } = useGames({ sortBy: 'modifiedAt', sortDir: 'desc' });

  const recentGames = useMemo(() => {
    if (!games) return [];
    return games.slice(0, 5);
  }, [games]);

  const items: ListItem[] = recentGames.map((game) => ({
    id: game.id ?? '',
    label: game.name ?? 'Untitled',
    sublabel: formatRelativeTime(game.meta?.modifiedAt ?? game.meta?.createdAt),
    onClick: () => navigate({ to: `/my-games/${game.id}` as '/' }),
  }));

  return (
    <InformationalCard
      title={t('cards.myGames.title')}
      items={items}
      emptyMessage={t('cards.myGames.empty')}
      viewAllLabel={t('cards.myGames.viewAll')}
      onViewAll={() => navigate({ to: ROUTES.MY_GAMES as '/' })}
      isLoading={isLoading}
    />
  );
}

// My Rooms Card - placeholder for rooms
function MyRoomsCard() {
  const { t } = useTranslation('dashboard');
  const navigate = useNavigate();

  const items: ListItem[] = [];

  return (
    <InformationalCard
      title={t('cards.myRooms.title')}
      items={items}
      emptyMessage={t('cards.myRooms.empty')}
      viewAllLabel={t('cards.myRooms.viewAll')}
      onViewAll={() => navigate({ to: '/rooms' as '/' })}
      isLoading={false}
    />
  );
}

// Last Played Card - shows recent play sessions
function LastPlayedCard() {
  const { t } = useTranslation('dashboard');
  const navigate = useNavigate();
  const { data: sessions, isLoading } = useUserSessions();

  const recentSessions = useMemo(() => {
    if (!sessions) return [];
    return sessions.slice(0, 5);
  }, [sessions]);

  const items: ListItem[] = recentSessions.map((session) => ({
    id: session.id ?? '',
    label: session.gameName ?? 'Untitled Game',
    sublabel: formatRelativeTime(session.meta?.modifiedAt ?? session.meta?.createdAt),
    onClick: () => navigate({ to: `/sessions/${session.id}` as '/' }),
  }));

  return (
    <InformationalCard
      title={t('cards.lastPlayed.title')}
      items={items}
      emptyMessage={t('cards.lastPlayed.empty')}
      viewAllLabel={t('cards.lastPlayed.viewAll')}
      onViewAll={() => navigate({ to: ROUTES.ALL_GAMES as '/' })}
      isLoading={isLoading}
    />
  );
}

// External Links Card
function ExternalLinksCard() {
  const { t } = useTranslation('dashboard');

  const links: LinkItem[] = [
    {
      id: EXTERNAL_LINKS.CHATGAMELAB.id,
      title: t('cards.externalLinks.mainSite.title'),
      description: t('cards.externalLinks.mainSite.description'),
      href: EXTERNAL_LINKS.CHATGAMELAB.href,
      icon: <IconSchool size={16} />,
    },
    {
      id: EXTERNAL_LINKS.JFF.id,
      title: t('cards.externalLinks.jff.title'),
      description: t('cards.externalLinks.jff.description'),
      href: EXTERNAL_LINKS.JFF.href,
      icon: <IconUsers size={16} />,
    },
  ];

  return (
    <LinksCard
      title={t('cards.externalLinks.title')}
      links={links}
      highlighted
    />
  );
}

// Quick Actions Card
function QuickActionsCard() {
  const { t } = useTranslation('dashboard');
  const navigate = useNavigate();

  const actions = [
    {
      id: 'start-new-game',
      label: t('quickActions.startNewGame'),
      icon: <IconDeviceGamepad2 size={16} />,
      onClick: () => navigate({ to: ROUTES.ALL_GAMES as '/' }),
    },  
    {
      id: 'create-game',
      label: t('quickActions.createNewGame'),
      icon: <IconTools size={16} />,
      onClick: () => navigate({ to: ROUTES.MY_GAMES + '/create' as '/' }),
    },
    {
      id: 'create-room',
      label: t('quickActions.createRoom'),
      icon: <IconBuilding size={16} />,
      onClick: () => navigate({ to: '/rooms' as '/' }),
      disabled: true
    },
    {
      id: 'invite-members',
      label: t('quickActions.inviteMembers'),
      icon: <IconUsers size={16} />,
      onClick: () => {},
      disabled: true,
    },
  ];

  return (
    <QuickActionCard
      title={t('quickActions.title')}
      actions={actions}
    />
  );
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
    </Stack>
  );
}

/**
 * Full Dashboard page with layout - for standalone use
 */
export function Dashboard() {
  const { t } = useTranslation('navigation');

  const navItems: NavItem[] = [
    { label: t('play'), icon: <IconPlayerPlay size={18} />, onClick: () => {} },
    { label: t('create'), icon: <IconEdit size={18} />, onClick: () => {} },
    { label: t('rooms'), icon: <IconBuilding size={18} />, onClick: () => {} },
    { label: t('groups'), icon: <IconUsers size={18} />, onClick: () => {} },
  ];

  return (
    <AppLayout 
      variant="authenticated" 
      navItems={navItems}
      headerProps={{
        onNotificationsClick: () => navigationLogger.debug('Notifications clicked'),
        onSettingsClick: () => navigationLogger.debug('Settings clicked'),
        onProfileClick: () => navigationLogger.debug('Profile clicked'),
      }}
    >
      <DashboardContent />
    </AppLayout>
  );
}
