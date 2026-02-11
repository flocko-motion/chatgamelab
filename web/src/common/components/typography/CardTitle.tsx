import { Title } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * CardTitle - Heading for a card component
 * 
 * USE WHEN:
 * - Card headers (Recent Activity, Quick Actions, Feature cards)
 * - Panel headings
 * 
 * DO NOT USE FOR:
 * - Page headings (use PageTitle)
 * - Section headings outside cards (use SectionTitle)
 * 
 * @example
 * <CardTitle>Recent Activity</CardTitle>
 * <CardTitle accent>Create Adventures</CardTitle>
 */

export interface CardTitleProps {
  children: ReactNode;
  accent?: boolean;
}

export function CardTitle({ children, accent = false }: CardTitleProps) {
  return (
    <Title order={3} c={accent ? 'accent.9' : 'gray.9'}>
      {children}
    </Title>
  );
}
