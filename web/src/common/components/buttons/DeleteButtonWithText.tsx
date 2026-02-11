import { Button, type ButtonProps } from '@mantine/core';
import { IconTrash } from '@tabler/icons-react';

export interface DeleteButtonWithTextProps extends Omit<ButtonProps, 'leftSection' | 'variant' | 'color'> {
  onClick?: () => void;
}

/**
 * DeleteButtonWithText - Button with delete icon and customizable text
 * 
 * USE WHEN:
 * - Delete actions that need descriptive text (e.g., "Delete Game", "Delete Session")
 * 
 * @example
 * <DeleteButtonWithText onClick={handleDelete}>Delete Game</DeleteButtonWithText>
 */
export function DeleteButtonWithText({
  children,
  size = 'xs',
  ...props
}: DeleteButtonWithTextProps) {
  return (
    <Button
      variant="subtle"
      color="red"
      size={size}
      radius="md"
      leftSection={<IconTrash size={14} />}
      {...props}
    >
      {children}
    </Button>
  );
}
