import { createFileRoute } from '@tanstack/react-router';
import { Stack, Title, Text } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { SettingsForm } from '@/features/settings';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute(ROUTES.SETTINGS)({
  component: SettingsPage,
});

function SettingsPage() {
  const { t } = useTranslation('auth');

  return (
    <Stack gap="xl">
      <Stack gap="xs">
        <Title order={1}>{t('settings.title')}</Title>
        <Text c="dimmed">{t('settings.subtitle')}</Text>
      </Stack>

      <SettingsForm />
    </Stack>
  );
}
