import { Card, Stack } from '@mantine/core';
import type { ReactNode } from 'react';
import { CardTitle } from '@components/typography';
import { MenuButton } from '@components/buttons';

interface QuickAction {
  id: string;
  label: string;
  icon?: ReactNode;
  onClick: () => void;
  disabled?: boolean;
}

interface QuickActionCardProps {
  title: string;
  actions: QuickAction[];
}

export function QuickActionCard({ title, actions }: QuickActionCardProps) {
  return (
    <Card p="lg" withBorder shadow="sm" h="100%">
      <CardTitle>{title}</CardTitle>
      <Stack gap="md" mt="md">
        {actions.map((action) => (
          <MenuButton
            key={action.id}
            leftSection={action.icon}
            onClick={action.onClick}
            disabled={action.disabled}
          >
            {action.label}
          </MenuButton>
        ))}
      </Stack>
    </Card>
  );
}
