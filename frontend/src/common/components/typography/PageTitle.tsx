import { Title } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * PageTitle - Main heading for a page
 * 
 * USE WHEN:
 * - Top-level heading on a page (Settings, Dashboard, Profile)
 * - One per page maximum
 * 
 * DO NOT USE FOR:
 * - Section headings within a page (use SectionTitle)
 * - Card headings (use CardTitle)
 * 
 * @example
 * <PageTitle>Settings</PageTitle>
 */

export interface PageTitleProps {
  children: ReactNode;
}

export function PageTitle({ children }: PageTitleProps) {
  return (
    <Title order={1} c="gray.9">
      {children}
    </Title>
  );
}
