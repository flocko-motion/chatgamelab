import { Button as MantineButton } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * MenuButton - Action button for menus and action lists
 * 
 * USE WHEN:
 * - Quick action panels/sidebars (Create Game, Create Room, Invite Members)
 * - Action lists within cards
 * - Navigation-like buttons in secondary areas
 * 
 * DO NOT USE FOR:
 * - Primary page CTAs (use ActionButton)
 * - Subtle/link-like actions (use TextButton)
 * - Destructive actions (use DangerButton)
 * 
 * @example
 * <MenuButton leftSection={<IconPlus />}>Create New Game</MenuButton>
 */

export interface MenuButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  disabled?: boolean;
}

export function MenuButton({
  children,
  onClick,
  leftSection,
  rightSection,
  disabled = false,
}: MenuButtonProps) {
  return (
    <MantineButton
      color="accent"
      variant="light"
      size="md"
      radius="md"
      onClick={onClick}
      leftSection={leftSection}
      rightSection={rightSection}
      disabled={disabled}
      fullWidth
      justify="start"
      styles={{
        label: {
          color: 'var(--mantine-color-accent-9)',
        },
      }}
    >
      {children}
    </MantineButton>
  );
}
