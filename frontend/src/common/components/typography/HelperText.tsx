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
  c?: string;
  ta?: 'left' | 'center' | 'right';
  /** At most this many lines, cut with an ellipsis. */
  lineClamp?: number;
  /** One line, cut with an ellipsis. */
  truncate?: boolean;
  size?: 'xs' | 'sm';
}

export function HelperText({ children, c, ta, lineClamp, truncate, size = 'sm' }: HelperTextProps) {
  return (
    <Text size={size} c={c || "gray.5"} ta={ta} lineClamp={lineClamp} truncate={truncate}>
      {children}
    </Text>
  );
}
