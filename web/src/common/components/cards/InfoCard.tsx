import { Alert, Stack, Text } from '@mantine/core';
import { IconInfoCircle } from '@tabler/icons-react';
import type { ReactNode } from 'react';

/**
 * InfoCard - Informational alert card for explanations and tips
 * 
 * USE WHEN:
 * - Explaining a feature or section to the user
 * - Providing helpful tips or context
 * - Showing non-critical informational messages
 * 
 * DO NOT USE FOR:
 * - Error messages (use Alert with color="red")
 * - Success messages (use notifications)
 * - Warning messages that require action
 * 
 * @example
 * <InfoCard title="About API Keys">
 *   API keys are used to authenticate with AI platforms.
 * </InfoCard>
 */

export interface InfoCardProps {
  title?: string;
  children: ReactNode;
  icon?: ReactNode;
}

export function InfoCard({ title, children, icon }: InfoCardProps) {
  return (
    <Alert 
      icon={icon ?? <IconInfoCircle size={18} />} 
      color="accent" 
      variant="light"
    >
      <Stack gap="xs">
        {title && <Text fw={600} size="sm">{title}</Text>}
        <Text size="sm">{children}</Text>
      </Stack>
    </Alert>
  );
}
