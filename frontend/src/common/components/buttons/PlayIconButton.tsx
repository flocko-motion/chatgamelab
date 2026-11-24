import { ActionIcon } from '@mantine/core';
import { IconPlayerPlay } from '@tabler/icons-react';
import { forwardRef } from 'react';
import type { IconButtonProps } from './types';

/**
 * PlayIconButton - Prominent icon button for play/start actions
 * 
 * USE WHEN:
 * - Starting a game or session
 * - Primary play action
 * 
 * Features a gradient background using accent (cyan) and highlight (magenta) colors
 * for maximum visual prominence as the primary action.
 * 
 * @example
 * <PlayIconButton onClick={handlePlay} aria-label="Play game" />
 */

export const PlayIconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function PlayIconButton({ onClick, 'aria-label': ariaLabel, disabled = false, loading = false, size = 'md' }, ref) {
    return (
      <ActionIcon
        ref={ref}
        variant="filled"
        color="accent"
        size={size}
        radius="md"
        onClick={onClick}
        disabled={disabled}
        loading={loading}
        aria-label={ariaLabel}
      >
        <IconPlayerPlay style={{ width: '70%', height: '70%' }} stroke={2} />
      </ActionIcon>
    );
  }
);
