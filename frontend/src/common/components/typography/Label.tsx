import { Text } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * Label - Text label for form fields or UI elements
 * 
 * USE WHEN:
 * - Form field labels
 * - UI element labels (stats, metadata)
 * - Small uppercase category labels
 * 
 * DO NOT USE FOR:
 * - Body text (use BodyText)
 * - Error messages (use ErrorText)
 * - Helper/description text (use HelperText)
 * 
 * @example
 * <Label>Email Address</Label>
 * <Label uppercase>Active Adventures</Label>
 */

export interface LabelProps {
  children: ReactNode;
  uppercase?: boolean;
}

export function Label({ children, uppercase = false }: LabelProps) {
  return (
    <Text 
      size="sm" 
      fw={600} 
      c="gray.5"
      tt={uppercase ? 'uppercase' : undefined}
    >
      {children}
    </Text>
  );
}
