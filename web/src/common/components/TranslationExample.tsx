import { Card, Title, Text, Button, Group, Stack } from '@mantine/core';
import { useBackendTranslation } from '../hooks/useTranslation';

export function TranslationExample() {
  const { t, isLoading, isBackendLanguage } = useBackendTranslation('common');
  const { t: tNav } = useBackendTranslation('navigation');
  const { t: tGame } = useBackendTranslation('game');

  return (
    <Card shadow="sm" padding="lg" radius="md" withBorder>
      <Stack gap="md">
        <Title order={3}>Translation Demo</Title>
        
        {isLoading && (
          <Text c="blue" size="sm">
            {isBackendLanguage ? 'Loading translations from backend...' : 'Loading translations...'}
          </Text>
        )}

        <Group gap="xs">
          <Text fw={500}>Common translations:</Text>
          <Text>{t('welcome')}</Text>
          <Text>•</Text>
          <Text>{t('hello')}</Text>
          <Text>•</Text>
          <Text>{t('goodbye')}</Text>
        </Group>

        <Group gap="xs">
          <Text fw={500}>Navigation:</Text>
          <Text>{tNav('home')}</Text>
          <Text>•</Text>
          <Text>{tNav('games')}</Text>
          <Text>•</Text>
          <Text>{tNav('create')}</Text>
        </Group>

        <Group gap="xs">
          <Text fw={500}>Game actions:</Text>
          <Text>{tGame('start')}</Text>
          <Text>•</Text>
          <Text>{tGame('pause')}</Text>
          <Text>•</Text>
          <Text>{tGame('score')}</Text>
        </Group>

        <Group gap="xs">
          <Text fw={500}>Form actions:</Text>
          <Button size="xs">{t('save')}</Button>
          <Button size="xs" variant="outline">{t('cancel')}</Button>
          <Button size="xs" variant="light">{t('delete')}</Button>
        </Group>

        {isBackendLanguage && (
          <Text c="orange" size="sm" fs="italic">
            🌐 These translations are loaded from the backend API
          </Text>
        )}
      </Stack>
    </Card>
  );
}
