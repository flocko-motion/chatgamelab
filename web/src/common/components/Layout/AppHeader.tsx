import {
  AppShell,
  Group,
  Box,
  Burger,
  ActionIcon,
  Drawer,
  Stack,
  Divider,
  Image,
  UnstyledButton,
  Text,
  useMantineTheme,
} from '@mantine/core';
import { useDisclosure, useElementSize } from '@mantine/hooks';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from '@tanstack/react-router';
import { useResponsiveDesign } from '../../hooks/useResponsiveDesign';
import { useAuth } from '../../../providers/AuthProvider';
import {
  IconSettings,
  IconBell,
  IconUser,
  IconLogout,
  IconKey,
} from '@tabler/icons-react';
import { LanguageSwitcher } from '../LanguageSwitcher';
import { DropdownMenu } from '../DropdownMenu';
import { UserAvatar } from '../UserAvatar';
import { getUserAvatarColor } from '@/common/lib/userUtils';
import { ROUTES } from '../../routes/routes';
import logoLandscapeWhite from '@/assets/logos/colorful/ChatGameLab-Logo-2025-Landscape-Colorful-White-Text-Transparent.png';

export interface NavItem {
  label: string;
  icon: React.ReactNode;
  onClick?: () => void;
  href?: string;
  active?: boolean;
}

export interface AppHeaderProps {
  navItems?: NavItem[];
  onNotificationsClick?: () => void;
  onSettingsClick?: () => void;
  onProfileClick?: () => void;
  onApiKeysClick?: () => void;
  onLogoutClick?: () => void;
  userName?: string;
}

function NavButton({ item }: { item: NavItem }) {
  const theme = useMantineTheme();

  return (
    <UnstyledButton
      onClick={item.onClick}
      py="xs"
      px="md"
      style={{
        borderRadius: 'var(--mantine-radius-md)',
        color: 'white',
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        transition: 'background-color 150ms ease, box-shadow 150ms ease',
      }}
      styles={{
        root: {
          backgroundColor: item.active ? theme.other.layout.bgActive : 'transparent',
          boxShadow: item.active ? '0 0 0 1px rgba(255, 255, 255, 0.3)' : 'none',
          '&:hover': {
            backgroundColor: item.active ? theme.other.layout.bgActive : 'rgba(255, 255, 255, 0.2)',
            boxShadow: item.active ? '0 0 0 1px rgba(255, 255, 255, 0.3)' : 'none',
          },
        },
      }}
    >
      {item.icon}
      <Text size="sm" fw={500}>{item.label}</Text>
    </UnstyledButton>
  );
}

function DesktopNavigation({ items }: { items: NavItem[] }) {
  return (
    <Group gap="xs" wrap="nowrap">
      {items.map((item, index) => (
        <NavButton key={index} item={item} />
      ))}
    </Group>
  );
}

function UserActions({
  onNotificationsClick,
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
}: Pick<AppHeaderProps, 'onNotificationsClick' | 'onSettingsClick' | 'onProfileClick' | 'onApiKeysClick' | 'onLogoutClick'>) {
  const { t } = useTranslation('common');
  const { logout: authLogout, backendUser } = useAuth();
  const theme = useMantineTheme();

  const handleLogout = () => {
    authLogout();
    onLogoutClick?.();
  };

  const userAvatar = (
    <UnstyledButton
      aria-label={t('header.openUserMenu')}
      style={{
        borderRadius: 999,
        padding: 2,
        border: `2px solid ${theme.colors[getUserAvatarColor(backendUser?.name || 'User')]?.[6] || theme.colors.accent[6]}`,
        backgroundColor: theme.other.layout.bgSubtle,
        transition: 'background-color 150ms ease, border-color 150ms ease',
      }}
      styles={{
        root: {
          '&:hover': {
            backgroundColor: theme.other.layout.bgHover,
          },
          '&:active': {
            backgroundColor: theme.other.layout.bgActive,
          },
          '&:focusVisible': {
            outline: `2px solid ${theme.colors.accent[6]}`,
            outlineOffset: 2,
          },
        },
      }}
    >
      <UserAvatar
        name={backendUser?.name || 'User'}
        size="md"
        style={{
          backgroundColor: 'transparent',
        }}
      />
    </UnstyledButton>
  );

  const userMenuItems = [
    {
      key: 'profile',
      label: t('header.profile'),
      icon: <IconUser size={16} />,
      onClick: onProfileClick,
    },
    {
      key: 'settings',
      label: t('header.settings'),
      icon: <IconSettings size={16} />,
      onClick: onSettingsClick,
    },
    {
      key: 'apiKeys',
      label: t('header.apiKeys'),
      icon: <IconKey size={16} />,
      onClick: onApiKeysClick,
    },
    {
      key: 'logout',
      label: t('header.logout'),
      icon: <IconLogout size={16} />,
      onClick: handleLogout,
      danger: true,
    },
  ];

  return (
    <Group gap="sm" wrap="nowrap">
      <LanguageSwitcher size="sm" variant="compact" />
      <ActionIcon
        variant="subtle"
        size="lg"
        c="white"
        onClick={onNotificationsClick}
        aria-label={t('header.viewNotifications')}
        styles={{
          root: {
            '&:hover': { backgroundColor: theme.other.layout.bgHover },
            '&:active': { backgroundColor: theme.other.layout.bgActive },
          },
        }}
      >
        <IconBell size={20} />
      </ActionIcon>
      <DropdownMenu
        trigger={userAvatar}
        items={userMenuItems}
        position="bottom"
      />
    </Group>
  );
}

function MobileNavigation({
  items,
  onNotificationsClick,
  onSettingsClick,
  onLogoutClick,
  onClose,
}: {
  items: NavItem[];
  onNotificationsClick?: () => void;
  onSettingsClick?: () => void;
  onLogoutClick?: () => void;
  onClose: () => void;
}) {
  const { t } = useTranslation('common');
  const { logout: authLogout } = useAuth();
  const theme = useMantineTheme();

  const handleLogout = () => {
    authLogout();
    onLogoutClick?.();
    onClose();
  };

  return (
    <Box style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box style={{ flex: 1, overflowY: 'auto' }}>
        <Stack gap="xs" p="md">
          {items.map((item, index) => (
            <UnstyledButton
              key={index}
              onClick={() => {
                item.onClick?.();
                onClose();
              }}
              py="md"
              px="lg"
              style={{
                borderRadius: 'var(--mantine-radius-md)',
                color: 'white',
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
                transition: 'background-color 150ms ease, box-shadow 150ms ease',
                backgroundColor: item.active ? theme.other.layout.bgActive : theme.other.layout.bgSubtle,
                boxShadow: item.active ? '0 0 0 1px rgba(255, 255, 255, 0.3)' : 'none',
              }}
            >
              {item.icon}
              <Text size="lg" fw={500}>{item.label}</Text>
            </UnstyledButton>
          ))}
        </Stack>
      </Box>

      <Box p="md">
        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Stack gap={6}>
          <UnstyledButton
            onClick={() => {
              onNotificationsClick?.();
              onClose();
            }}
            py="sm"
            px="lg"
            style={{
              borderRadius: 'var(--mantine-radius-md)',
              color: 'white',
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
            }}
          >
            <IconBell size={20} />
            <Text size="md" fw={500}>{t('notifications')}</Text>
          </UnstyledButton>

          <UnstyledButton
            onClick={() => {
              onSettingsClick?.();
              onClose();
            }}
            py="sm"
            px="lg"
            style={{
              borderRadius: 'var(--mantine-radius-md)',
              color: 'white',
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
            }}
          >
            <IconSettings size={20} />
            <Text size="md" fw={500}>{t('settings')}</Text>
          </UnstyledButton>

          <UnstyledButton
            onClick={handleLogout}
            py="sm"
            px="lg"
            style={{
              borderRadius: 'var(--mantine-radius-md)',
              color: theme.colors.red[6],
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
            }}
          >
            <IconLogout size={20} />
            <Text size="md" fw={500}>{t('header.logout')}</Text>
          </UnstyledButton>
        </Stack>

        <Divider my="xs" color={theme.other.layout.lineLight} />

        <Box px="lg">
          <LanguageSwitcher size="sm" variant="compact" />
        </Box>
      </Box>
    </Box>
  );
}

export function AppHeader({
  navItems = [],
  onNotificationsClick,
  onSettingsClick,
  onProfileClick,
  onApiKeysClick,
  onLogoutClick,
}: AppHeaderProps) {
  const [mobileNavOpened, { open: openMobileNav, close: closeMobileNav }] = useDisclosure(false);
  const { isMobile: isViewportMobile } = useResponsiveDesign();
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const theme = useMantineTheme();

  // Measure header container width
  const { ref: headerRef, width: headerWidth } = useElementSize();
  const { ref: measureNavRef, width: navWidth } = useElementSize();
  const { ref: measureLogoRef, width: logoWidth } = useElementSize();
  const { ref: measureActionsRef, width: actionsWidth } = useElementSize();

  // With left-aligned layout, we only need to check if total content fits
  const contentOverflows = (() => {
    if (headerWidth === 0 || navWidth === 0 || logoWidth === 0 || actionsWidth === 0) return false;

    const padding = 32; // p="md" = 16px * 2
    const gap = 16; // gap="md" between left and right sections
    const totalWidth = logoWidth + navWidth + actionsWidth + gap + padding;

    return totalWidth > headerWidth;
  })();

  // Force mobile if viewport is mobile OR content overflows
  const forceMobile = isViewportMobile || contentOverflows;

  useEffect(() => {
    if (!forceMobile && mobileNavOpened) {
      closeMobileNav();
    }
  }, [forceMobile, mobileNavOpened, closeMobileNav]);

  return (
    <>
      {/* Hidden measurement container - always rendered to measure content */}
      <Box
        style={{
          position: 'absolute',
          visibility: 'hidden',
          pointerEvents: 'none',
          height: 0,
          overflow: 'hidden',
        }}
        aria-hidden="true"
      >
        <Box ref={measureNavRef} style={{ display: 'inline-block' }}>
          <DesktopNavigation items={navItems} />
        </Box>
        <Box ref={measureLogoRef} style={{ display: 'inline-block' }}>
          <Image
            src={logoLandscapeWhite}
            alt=""
            h={50}
            w="auto"
            fit="contain"
          />
        </Box>
        <Box ref={measureActionsRef} style={{ display: 'inline-block' }}>
          <UserActions
            onNotificationsClick={onNotificationsClick}
            onSettingsClick={onSettingsClick}
            onProfileClick={onProfileClick}
            onApiKeysClick={onApiKeysClick}
            onLogoutClick={onLogoutClick}
          />
        </Box>
      </Box>

      <AppShell.Header
        ref={headerRef}
        p="md"
        style={{
          background: theme.other.layout.headerGradient,
          borderBottom: theme.other.layout.borderLight,
          boxShadow: theme.other.layout.shadowHeader,
        }}
      >
        <Group justify="space-between" align="center" h="100%" wrap="nowrap" gap="md">
          {/* Left section: Logo + Navigation (desktop) */}
          <Group gap="lg" align="center" wrap="nowrap">
            {/* Logo */}
            <UnstyledButton
              onClick={() => navigate({ to: ROUTES.DASHBOARD })}
              aria-label={t('header.goToDashboard')}
              style={{
                borderRadius: 'var(--mantine-radius-sm)',
                transition: 'background-color 150ms ease',
              }}
              styles={{
                root: {
                  '&:hover': {
                    backgroundColor: theme.other.layout.bgHover,
                  },
                  '&:active': {
                    backgroundColor: theme.other.layout.bgActive,
                  },
                  '&:focusVisible': {
                    outline: `2px solid ${theme.colors.accent[6]}`,
                    outlineOffset: 2,
                  },
                },
              }}
            >
              <Image
                src={logoLandscapeWhite}
                alt="ChatGameLab Logo"
                h={{ base: 36, sm: 50 }}
                w="auto"
                fit="contain"
              />
            </UnstyledButton>

            {/* Desktop: Nav items */}
            {!forceMobile && (
              <DesktopNavigation items={navItems} />
            )}
          </Group>

          {/* Right section */}
          <Group gap="sm" align="center" wrap="nowrap">
            {/* Desktop: Full user actions */}
            {!forceMobile && (
              <UserActions
                onNotificationsClick={onNotificationsClick}
                onSettingsClick={onSettingsClick}
                onProfileClick={onProfileClick}
                onApiKeysClick={onApiKeysClick}
                onLogoutClick={onLogoutClick}
              />
            )}

            {/* Mobile: Burger menu */}
            {forceMobile && (
              <Burger
                opened={mobileNavOpened}
                onClick={openMobileNav}
                color="white"
                size="sm"
                aria-label={mobileNavOpened ? t('header.closeNavigation') : t('header.openNavigation')}
              />
            )}
          </Group>
        </Group>
      </AppShell.Header>

      {/* Mobile Navigation Drawer */}
      <Drawer
        opened={mobileNavOpened}
        onClose={closeMobileNav}
        size="100%"
        withCloseButton
        styles={{
          header: {
            background: theme.other.layout.headerGradient,
            borderBottom: theme.other.layout.borderLight,
          },
          close: {
            color: 'white',
            '&:hover': {
              backgroundColor: theme.other.layout.bgHover,
            },
          },
          content: {
            background: theme.other.layout.drawerGradient,
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
          },
          body: {
            padding: 0,
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            minHeight: 0,
          },
        }}
        title={
          <Image
            src={logoLandscapeWhite}
            alt="ChatGameLab Logo"
            h={40}
            w="auto"
            fit="contain"
          />
        }
      >
        <MobileNavigation
          items={navItems}
          onNotificationsClick={onNotificationsClick}
          onSettingsClick={onSettingsClick}
          onLogoutClick={onLogoutClick}
          onClose={closeMobileNav}
        />
      </Drawer>
    </>
  );
}
