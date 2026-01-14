import { Button as MantineButton } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * DangerButton - Button for destructive or irreversible actions
 * 
 * USE WHEN:
 * - Delete/remove actions
 * - Destructive operations that cannot be undone
 * - Error recovery actions ("Try Again" after crash)
 * - Actions that require extra user attention due to consequences
 * 
 * DO NOT USE FOR:
 * - Primary actions (use ActionButton)
 * - Cancel/dismiss (use TextButton)
 * - Non-destructive actions
 * 
 * @example
 * <DangerButton onClick={handleDelete}>Delete Account</DangerButton>
 * <DangerButton variant="outline" onClick={handleRetry}>Try Again</DangerButton>
 */

export interface DangerButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  loading?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
  variant?: 'filled' | 'outline';
  fullWidth?: boolean;
}

export function DangerButton({
  children,
  onClick,
  leftSection,
  rightSection,
  loading = false,
  disabled = false,
  type = 'button',
  variant = 'filled',
  fullWidth = false,
}: DangerButtonProps) {
  return (
    <MantineButton
      color="red"
      variant={variant}
      size="md"
      radius="md"
      onClick={onClick}
      leftSection={leftSection}
      rightSection={rightSection}
      loading={loading}
      disabled={disabled}
      type={type}
      fullWidth={fullWidth}
    >
      {children}
    </MantineButton>
  );
}
