import { Text } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * BodyText - Standard body/paragraph text
 * 
 * USE WHEN:
 * - Main content paragraphs
 * - Descriptions
 * - Any primary readable content
 * 
 * DO NOT USE FOR:
 * - Headings (use PageTitle, SectionTitle, CardTitle)
 * - Labels (use Label)
 * - Helper/muted text (use HelperText)
 * - Errors (use ErrorText)
 * 
 * @example
 * <BodyText>An educational platform for creating AI-powered games.</BodyText>
 * <BodyText size="lg">Larger body text for hero sections.</BodyText>
 */

export interface BodyTextProps {
  children: ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  fw?: number;
  c?: string;
  component?: 'span' | 'p' | 'div';
}

export function BodyText({ children, size = 'md', fw, c, component }: BodyTextProps) {
  if (component === 'span') {
    return (
      <Text size={size} c={c || "gray.7"} lh={1.6} fw={fw} component="span">
        {children}
      </Text>
    );
  }
  return (
    <Text size={size} c={c || "gray.7"} lh={1.6} fw={fw}>
      {children}
    </Text>
  );
}
