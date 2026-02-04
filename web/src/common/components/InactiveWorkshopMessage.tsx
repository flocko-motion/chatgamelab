import { Stack, Text, Card, Center, Button } from '@mantine/core';
import { IconSchoolOff } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

interface InactiveWorkshopMessageProps {
  onLogout: () => void;
}

export function InactiveWorkshopMessage({ onLogout }: InactiveWorkshopMessageProps) {
  const { t } = useTranslation('common');

  return (
    <Center h="60vh">
      <Card shadow="md" p="xl" radius="md" withBorder maw={500} w="100%">
        <Stack align="center" gap="lg">
          <IconSchoolOff size={64} color="var(--mantine-color-orange-5)" />
          <Text size="xl" fw={600} ta="center">
            {t('workshop.inactive.title', 'Workshop Inactive')}
          </Text>
          <Text c="dimmed" ta="center">
            {t('workshop.inactive.description', 'This workshop is currently not active. Please wait until the workshop organizer enables it again, or log out if you want to leave.')}
          </Text>
          <Button
            variant="outline"
            color="gray"
            onClick={onLogout}
            mt="md"
          >
            {t('logout', 'Log Out')}
          </Button>
        </Stack>
      </Card>
    </Center>
  );
}
