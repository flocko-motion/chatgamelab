import {
  Group,
  Stack,
  Text,
  UnstyledButton,
  useMantineTheme,
  Popover,
  Divider,
  Box,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconDoorExit, IconSchool, IconBuilding } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../../../providers/AuthProvider';
import { UserAvatar } from '../UserAvatar';
import { getUserAvatarColor } from '@/common/lib/userUtils';

interface ParticipantUserMenuProps {
  workshopName?: string;
  organizationName?: string;
}

/**
 * Simplified user menu for workshop participants.
 * Shows name, workshop, organization info and logout only.
 */
export function ParticipantUserMenu({
  workshopName,
  organizationName,
}: ParticipantUserMenuProps) {
  const { t } = useTranslation('common');
  const { t: tParticipant } = useTranslation('participant');
  const { logout, backendUser } = useAuth();
  const theme = useMantineTheme();
  const [opened, { close, toggle }] = useDisclosure(false);

  const userName = backendUser?.name || 'Participant';

  return (
    <Popover
      opened={opened}
      onClose={close}
      position="bottom-end"
      withArrow
      shadow="md"
      width={280}
    >
      <Popover.Target>
        <UnstyledButton
          onClick={toggle}
          aria-label={t('header.openUserMenu')}
          style={{
            borderRadius: 999,
            padding: 2,
            border: `2px solid ${theme.colors[getUserAvatarColor(userName)]?.[6] || theme.colors.accent[6]}`,
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
            },
          }}
        >
          <UserAvatar
            name={userName}
            size="md"
            style={{
              backgroundColor: 'transparent',
            }}
          />
        </UnstyledButton>
      </Popover.Target>

      <Popover.Dropdown p="md">
        <Stack gap="sm">
          {/* User info section */}
          <Group gap="sm" wrap="nowrap">
            <UserAvatar name={userName} size="lg" />
            <Box style={{ flex: 1, minWidth: 0 }}>
              <Text fw={600} size="sm" truncate>
                {userName}
              </Text>
              <Text size="xs" c="dimmed">
                {tParticipant('role')}
              </Text>
            </Box>
          </Group>

          <Divider />

          {/* Workshop info */}
          {workshopName && (
            <Group gap="xs" wrap="nowrap">
              <IconSchool size={16} style={{ color: theme.colors.accent[6], flexShrink: 0 }} />
              <Box style={{ flex: 1, minWidth: 0 }}>
                <Text size="xs" c="dimmed">{tParticipant('workshop')}</Text>
                <Text size="sm" fw={500} truncate>{workshopName}</Text>
              </Box>
            </Group>
          )}

          {/* Organization info */}
          {organizationName && (
            <Group gap="xs" wrap="nowrap">
              <IconBuilding size={16} style={{ color: theme.colors.violet[6], flexShrink: 0 }} />
              <Box style={{ flex: 1, minWidth: 0 }}>
                <Text size="xs" c="dimmed">{tParticipant('organization')}</Text>
                <Text size="sm" fw={500} truncate>{organizationName}</Text>
              </Box>
            </Group>
          )}

          <Divider />

          {/* Leave workshop button */}
          <UnstyledButton
            onClick={() => {
              close();
              logout();
            }}
            py="xs"
            px="sm"
            style={{
              borderRadius: 'var(--mantine-radius-sm)',
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              color: theme.colors.red[6],
              transition: 'background-color 150ms ease',
            }}
            styles={{
              root: {
                '&:hover': {
                  backgroundColor: theme.colors.red[0],
                },
              },
            }}
          >
            <IconDoorExit size={16} />
            <Text size="sm" fw={500}>{tParticipant('leaveWorkshop')}</Text>
          </UnstyledButton>
        </Stack>
      </Popover.Dropdown>
    </Popover>
  );
}
