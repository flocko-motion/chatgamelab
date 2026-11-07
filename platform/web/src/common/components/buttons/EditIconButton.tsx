import { ActionIcon } from '@mantine/core';
import { IconEdit } from '@tabler/icons-react';
import { forwardRef } from 'react';
import type { IconButtonProps } from './types';

/**
 * EditIconButton - Specialized icon button for edit actions
 * 
 * USE WHEN:
 * - Editing items in lists or tables
 * - Modifying entries
 * 
 * @example
 * <EditIconButton onClick={handleEdit} aria-label="Edit item" />
 */

export const EditIconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function EditIconButton({ onClick, 'aria-label': ariaLabel, disabled = false, loading = false, size = 'md' }, ref) {
    return (
      <ActionIcon
        ref={ref}
        variant="subtle"
        color="blue"
        size={size}
        radius="md"
        onClick={onClick}
        disabled={disabled}
        loading={loading}
        aria-label={ariaLabel}
      >
        <IconEdit style={{ width: '70%', height: '70%' }} stroke={1.5} />
      </ActionIcon>
    );
  }
);
