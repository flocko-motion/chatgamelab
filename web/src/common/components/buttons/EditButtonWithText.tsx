import { Button, type ButtonProps } from '@mantine/core';
import { IconPencil } from '@tabler/icons-react';

export interface EditButtonWithTextProps extends Omit<ButtonProps, 'leftSection' | 'variant' | 'color'> {
  onClick?: () => void;
}

/**
 * EditButtonWithText - Button with edit icon and customizable text
 * 
 * USE WHEN:
 * - Edit actions that need descriptive text (e.g., "Edit Game", "Edit Profile")
 * 
 * @example
 * <EditButtonWithText onClick={handleEdit}>Edit Game</EditButtonWithText>
 */
export function EditButtonWithText({
  children,
  size = 'xs',
  ...props
}: EditButtonWithTextProps) {
  return (
    <Button
      variant="subtle"
      color="blue"
      size={size}
      radius="md"
      leftSection={<IconPencil size={14} />}
      {...props}
    >
      {children}
    </Button>
  );
}
