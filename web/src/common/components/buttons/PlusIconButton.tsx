import { ActionIcon } from '@mantine/core';
import { IconPlus } from '@tabler/icons-react';
import type { IconButtonProps } from './types';

/**
 * PlusIconButton - Icon button for add/create actions
 * 
 * USE WHEN:
 * - Adding new items to a list
 * - Creating new entries
 * - Expanding/adding content
 * 
 * @example
 * <PlusIconButton onClick={handleAdd} aria-label="Add item" />
 */

export interface PlusIconButtonProps extends IconButtonProps {
  variant?: 'subtle' | 'filled' | 'light' | 'outline' | 'default';
}

export function PlusIconButton({
  onClick,
  'aria-label': ariaLabel,
  disabled = false,
  loading = false,
  size = 'md',
  variant = 'light',
}: PlusIconButtonProps) {
  const iconSize = size === 'xs' ? 12 : size === 'sm' ? 14 : 16;
  
  return (
    <ActionIcon
      variant={variant}
      color="violet"
      size={size}
      radius="md"
      onClick={onClick}
      disabled={disabled}
      loading={loading}
      aria-label={ariaLabel}
    >
      <IconPlus size={iconSize} />
    </ActionIcon>
  );
}
