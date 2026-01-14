import { Button as MantineButton } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * TextButton - Subtle, link-like button for secondary actions
 * 
 * USE WHEN:
 * - Secondary navigation ("View All", "See More", "Learn More")
 * - Cancel/dismiss actions in forms or modals
 * - Low-emphasis actions that shouldn't compete with primary CTAs
 * - Inline actions within content
 * 
 * DO NOT USE FOR:
 * - Primary actions (use ActionButton)
 * - Menu/list actions (use MenuButton)
 * - Destructive actions (use DangerButton)
 * 
 * @example
 * <TextButton onClick={handleViewAll}>View All</TextButton>
 * <TextButton onClick={handleCancel}>Cancel</TextButton>
 */

export interface TextButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  disabled?: boolean;
  size?: 'xs' | 'sm' | 'md';
}

export function TextButton({
  children,
  onClick,
  leftSection,
  rightSection,
  disabled = false,
  size = 'sm',
}: TextButtonProps) {
  return (
    <MantineButton
      variant="subtle"
      color="accent"
      size={size}
      radius="md"
      onClick={onClick}
      leftSection={leftSection}
      rightSection={rightSection}
      disabled={disabled}
    >
      {children}
    </MantineButton>
  );
}
