import { createFileRoute } from '@tanstack/react-router';
import { Stack, Title } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { ProfileView } from '@/features/profile';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute(ROUTES.PROFILE)({
  component: ProfilePage,
});

function ProfilePage() {
  const { t } = useTranslation('auth');

  return (
    <Stack gap="xl">
      <Title order={1}>{t('profile.title')}</Title>
      <ProfileView />
    </Stack>
  );
}
