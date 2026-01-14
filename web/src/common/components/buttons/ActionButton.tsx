import { Button as MantineButton } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * ActionButton - Primary call-to-action button
 * 
 * USE WHEN:
 * - Main action on a page (Login, Sign Up, Get Started, Submit)
 * - Hero section CTAs
 * - Form submission buttons
 * - Any action that is THE primary thing the user should do
 * 
 * DO NOT USE FOR:
 * - Secondary actions (use TextButton)
 * - Menu/list actions (use MenuButton)
 * - Destructive actions (use DangerButton)
 * 
 * @example
 * <ActionButton onClick={handleLogin}>Get Started</ActionButton>
 * <ActionButton leftSection={<IconRocket />} loading={isLoading}>Submit</ActionButton>
 */

export interface ActionButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  loading?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
  fullWidth?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

export function ActionButton({
  children,
  onClick,
  leftSection,
  rightSection,
  loading = false,
  disabled = false,
  type = 'button',
  fullWidth = false,
  size = 'lg',
}: ActionButtonProps) {
  return (
    <MantineButton
      color="accent"
      variant="filled"
      size={size}
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
