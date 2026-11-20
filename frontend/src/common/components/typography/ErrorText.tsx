import { Text } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * ErrorText - Error message text
 * 
 * USE WHEN:
 * - Form validation errors
 * - API error messages
 * - Any error state communication
 * 
 * DO NOT USE FOR:
 * - Warnings (use different styling)
 * - Helper text (use HelperText)
 * - General muted text
 * 
 * @example
 * <ErrorText>Email is required</ErrorText>
 */

export interface ErrorTextProps {
  children: ReactNode;
}

export function ErrorText({ children }: ErrorTextProps) {
  return (
    <Text size="sm" c="red">
      {children}
    </Text>
  );
}
