import { Text } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * HelperText - Secondary descriptive text
 * 
 * USE WHEN:
 * - Form field descriptions
 * - Supplementary information below a heading
 * - Muted context/hints
 * 
 * DO NOT USE FOR:
 * - Error messages (use ErrorText)
 * - Primary body text (use BodyText)
 * - Labels (use Label)
 * 
 * @example
 * <HelperText>Enter your email to receive updates</HelperText>
 */

export interface HelperTextProps {
  children: ReactNode;
}

export function HelperText({ children }: HelperTextProps) {
  return (
    <Text size="sm" c="gray.5">
      {children}
    </Text>
  );
}
