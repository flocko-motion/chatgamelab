import { Button, type ButtonProps } from '@mantine/core';
import type { ReactNode } from 'react';

export interface GenericButtonWithTextProps extends Omit<ButtonProps, 'leftSection'> {
  icon: ReactNode;
  onClick?: () => void;
}

/**
 * GenericButtonWithText - Button with icon and customizable text
 * 
 * USE WHEN:
 * - You need a consistent button style with custom icon and text
 * - Actions that need both visual icon and descriptive text
 * 
 * @example
 * <GenericButtonWithText 
 *   icon={<IconStar size={14} />} 
 *   onClick={handleFavorite}
 * >
 *   Add to favorites
 * </GenericButtonWithText>
 */
export function GenericButtonWithText({
  icon,
  children,
  size = 'xs',
  variant = 'subtle',
  color = 'gray',
  ...props
}: GenericButtonWithTextProps) {
  return (
    <Button
      variant={variant}
      color={color}
      size={size}
      radius="md"
      leftSection={icon}
      {...props}
    >
      {children}
    </Button>
  );
}
