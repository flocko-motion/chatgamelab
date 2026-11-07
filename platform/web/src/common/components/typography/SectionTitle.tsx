import { Title } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * SectionTitle - Heading for a section within a page
 * 
 * USE WHEN:
 * - Grouping related content (Features, Recent Activity, Quick Actions)
 * - Visual separation of page areas
 * 
 * DO NOT USE FOR:
 * - Top-level page heading (use PageTitle)
 * - Card headings (use CardTitle)
 * 
 * @example
 * <SectionTitle>Why Choose ChatGameLab?</SectionTitle>
 */

export interface SectionTitleProps {
  children: ReactNode;
  accent?: boolean;
}

export function SectionTitle({ children, accent = false }: SectionTitleProps) {
  return (
    <Title order={2} c={accent ? 'accent.9' : 'gray.9'}>
      {children}
    </Title>
  );
}
