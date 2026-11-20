import { Button, type ButtonProps } from '@mantine/core';

export interface CancelButtonProps extends Omit<ButtonProps, 'variant' | 'color'> {
  onClick?: () => void;
}

/**
 * CancelButton - Red button for cancel/abort actions
 * 
 * USE WHEN:
 * - Canceling a form or modal
 * - Aborting an operation
 * - Dismissing without saving
 * 
 * @example
 * <CancelButton onClick={handleCancel}>{t('cancel')}</CancelButton>
 */
export function CancelButton({
  children,
  size = 'sm',
  ...props
}: CancelButtonProps) {
  return (
    <Button
      variant="filled"
      color="red"
      size={size}
      radius="md"
      {...props}
    >
      {children}
    </Button>
  );
}
