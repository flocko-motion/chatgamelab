import { ActionIcon } from '@mantine/core';
import { forwardRef } from 'react';
import type { ReactNode } from 'react';
import type { IconButtonProps } from './types';

/**
 * GenericIconButton - Flexible icon button that accepts any icon
 * 
 * USE WHEN:
 * - You need a consistent icon button style with a custom icon
 * - Actions that don't have a specific semantic button component
 * 
 * @example
 * <GenericIconButton 
 *   icon={<IconStar size={16} />} 
 *   onClick={handleFavorite} 
 *   aria-label="Add to favorites" 
 * />
 */

export interface GenericIconButtonProps extends IconButtonProps {
  icon: ReactNode;
  color?: string;
  variant?: 'subtle' | 'filled' | 'light' | 'outline' | 'default';
}

export const GenericIconButton = forwardRef<HTMLButtonElement, GenericIconButtonProps>(
  function GenericIconButton(
    {
      icon,
      onClick,
      'aria-label': ariaLabel,
      disabled = false,
      loading = false,
      size = 'md',
      color = 'gray',
      variant = 'subtle',
    },
    ref
  ) {
    return (
      <ActionIcon
        ref={ref}
        variant={variant}
        color={color}
        size={size}
        radius="md"
        onClick={onClick}
        disabled={disabled}
        loading={loading}
        aria-label={ariaLabel}
      >
        {icon}
      </ActionIcon>
    );
  }
);
