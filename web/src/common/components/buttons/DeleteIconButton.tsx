import { ActionIcon } from '@mantine/core';
import { IconTrash } from '@tabler/icons-react';
import type { IconButtonProps } from './types';

/**
 * DeleteIconButton - Specialized icon button for delete actions
 * 
 * USE WHEN:
 * - Deleting items from lists or tables
 * - Removing entries
 * 
 * @example
 * <DeleteIconButton onClick={handleDelete} aria-label="Delete item" />
 */

export function DeleteIconButton({
  onClick,
  'aria-label': ariaLabel,
  disabled = false,
  loading = false,
  size = 'md',
}: IconButtonProps) {
  return (
    <ActionIcon
      variant="subtle"
      color="red"
      size={size}
      radius="md"
      onClick={onClick}
      disabled={disabled}
      loading={loading}
      aria-label={ariaLabel}
    >
      <IconTrash style={{ width: '70%', height: '70%' }} stroke={1.5} />
    </ActionIcon>
  );
}
